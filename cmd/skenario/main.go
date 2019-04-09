/*
 * Copyright (C) 2019-Present Pivotal Software, Inc. All rights reserved.
 *
 * This program and the accompanying materials are made available under the terms
 * of the Apache License, Version 2.0 (the "License”); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at:
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/knative/pkg/logging"
	"github.com/logrusorgru/aurora"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"skenario/pkg/model"

	"skenario/pkg/simulator"
)

var (
	startAt                     = time.Unix(0, 0)
	startRunning                = time.Now()
	au                          = aurora.NewAurora(true)
	simDuration                 = flag.Duration("duration", 10*time.Minute, "Duration of time to simulate.")
	tickInterval                = flag.Duration("tickInterval", 2*time.Second, "Tick interval duration of the Autoscaler")
	stableWindow                = flag.Duration("stableWindow", 60*time.Second, "Duration of stable window of the Autoscaler")
	panicWindow                 = flag.Duration("panicWindow", 6*time.Second, "Duration of panic window of the Autoscaler")
	scaleToZeroGrace            = flag.Duration("scaleToZeroGrace", 30*time.Second, "Duration of the scale-to-zero grace period of the Autoscaler")
	targetConcurrencyDefault    = flag.Float64("targetConcurrencyDefault", 1.0, "Default target concurrency of Replicas")
	targetConcurrencyPercentage = flag.Float64("targetConcurrencyPercentage", 0.5, "Percentage adjustment of target concurrency of Replicas")
	maxScaleUpRate              = flag.Float64("maxScaleUpRate", 10.0, "Maximum rate the autoscaler can raise its desired")
	launchDelay                 = flag.Duration("replicaLaunchDelay", time.Second, "Time it takes a Replica to move from launching to active")
	terminateDelay              = flag.Duration("replicaTerminateDelay", time.Second, "Time it takes a Replica to move from launching or active to terminated")
	numberOfRequests            = flag.Uint("numberOfRequests", 10, "Number of randomly-arriving requests to generate")
)

func main() {
	flag.Parse()
	r := NewRunner()

	cluster := model.NewCluster(r.Env(), r.ClusterConfig())
	model.NewKnativeAutoscaler(r.Env(), startAt, cluster, r.AutoscalerConfig())

	err := r.RunAndReport(os.Stdout)
	if err != nil {
		fmt.Printf("there was an error during simulation: %s", err.Error())
	}
}

type Runner interface {
	Env() simulator.Environment
	AutoscalerConfig() model.KnativeAutoscalerConfig
	ClusterConfig() model.ClusterConfig
	RunAndReport(writer io.Writer) error
}

type runner struct {
	env    simulator.Environment
	logbuf *bytes.Buffer
}

func (r *runner) RunAndReport(writer io.Writer) error {
	fmt.Fprint(writer, "Running simulation ... ")

	completed, ignored, err := r.env.Run()
	if err != nil {
		return err
	}

	fmt.Fprintf(writer,
		"%5s      %19s %-8d  %17s %-8d  %20s %-10s    %20s %-12s\n\n",
		au.Bold("Done."),
		au.BgGreen("Completed movements"),
		au.Bold(len(completed)),
		au.BgBrown("Ignored movements"),
		au.Bold(len(ignored)),
		au.Cyan("Running time:"),
		time.Now().Sub(startRunning).String(),
		au.Cyan("Simulated time:"),
		simDuration.String(),
	)

	printer := message.NewPrinter(language.AmericanEnglish)
	fmt.Fprintln(writer, au.BgGreen(fmt.Sprintf("%20s  %-24s %-14s %-34s ⟶   %-34s  %-58s", "Time (ns)", "Movement Name", "Entity Name", "From Stock", "To Stock", "Notes")).Bold())

	for _, c := range completed {
		mv := c.Movement
		e := c.Moved
		eName := "<nil>"
		if e != nil {
			eName = string(e.Name())
		}

		fmt.Fprintln(writer, printer.Sprintf(
			"%20d  %-24s %-14s %-34s ⟶   %-34s  %s",
			mv.OccursAt().UnixNano(),
			mv.Kind(),
			eName,
			mv.From().Name(),
			mv.To().Name(),
			strings.Join(mv.Notes(), fmt.Sprintf("\n%-137s", " ")),
		))
	}

	fmt.Fprint(writer, "\n")
	fmt.Fprintln(writer, au.BgBrown(fmt.Sprintf("%20s  %-24s %-14s %-34s ⟶   %-34s  %-28s %-29s", "Time (ns)", "Movement Name", "Entity Name", "From Stock", "To Stock", "Notes", "Reason Ignored")).Bold())
	for _, i := range ignored {
		mv := i.Movement

		coloredReason := ""
		switch i.Reason {
		case simulator.OccursInPast:
			coloredReason = au.Red(i.Reason).String()
		case simulator.OccursAfterHalt:
			coloredReason = au.Magenta(i.Reason).String()
		case simulator.OccursSimultaneouslyWithAnotherMovement:
			coloredReason = au.Cyan(i.Reason).String()
		case simulator.FromStockIsEmpty:
			coloredReason = au.Brown(i.Reason).String()
		}

		fmt.Fprintln(writer, printer.Sprintf(
			"%20d  %-24s %-14s %-34s ⟶   %-34s  %-28s %-29s",
			mv.OccursAt().UnixNano(),
			mv.Kind(),
			"-",
			mv.From().Name(),
			mv.To().Name(),
			strings.Join(mv.Notes(), fmt.Sprintf("\n%-137s", " ")),
			coloredReason,
		))
	}
	fmt.Fprint(writer, "\n")
	fmt.Fprintln(writer, au.Bold(fmt.Sprintf("%-195s", "          Log output from Knative")).BgBlue())
	fmt.Fprintln(writer, r.logbuf.String())

	return nil
}

func (r *runner) Env() simulator.Environment {
	return r.env
}

func (r *runner) AutoscalerConfig() model.KnativeAutoscalerConfig {
	return model.KnativeAutoscalerConfig{
		TickInterval:                *tickInterval,
		StableWindow:                *stableWindow,
		PanicWindow:                 *panicWindow,
		ScaleToZeroGracePeriod:      *scaleToZeroGrace,
		TargetConcurrencyDefault:    *targetConcurrencyDefault,
		TargetConcurrencyPercentage: *targetConcurrencyPercentage,
		MaxScaleUpRate:              *maxScaleUpRate,
	}
}

func (r *runner) ClusterConfig() model.ClusterConfig {
	return model.ClusterConfig{
		LaunchDelay:      *launchDelay,
		TerminateDelay:   *terminateDelay,
		NumberOfRequests: *numberOfRequests,
	}
}

func NewRunner() Runner {
	buf := new(bytes.Buffer)
	logger := newLogger(buf)
	ctx := logging.WithLogger(context.Background(), logger)

	return &runner{
		env:    simulator.NewEnvironment(ctx, startAt, *simDuration),
		logbuf: buf,
	}
}

func newLogger(buf io.Writer) *zap.SugaredLogger {
	sink := zapcore.AddSync(buf)

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		sink,
		zap.InfoLevel,
	)

	unsugaredLogger := zap.New(core)

	return unsugaredLogger.Named("skenario").Sugar()
}
