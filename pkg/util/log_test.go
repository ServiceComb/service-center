/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package util

import (
	"fmt"
	"testing"
)

func init() {
	InitGlobalLogger(LoggerConfig{
		LoggerLevel:   "DEBUG",
		LoggerFile:    "",
		LogFormatText: true,
	})
}

func TestLogger(t *testing.T) {
	CustomLogger("Not Exist", "testDefaultLOGGER")
	l := Logger()
	if l != LOGGER {
		fmt.Println("should equal to LOGGER")
		t.FailNow()
	}
	CustomLogger("TestLogger", "testFuncName")
	l = Logger()
	if l == LOGGER || l == nil {
		fmt.Println("should create a new instance for 'TestLogger'")
		t.FailNow()
	}
	s := Logger()
	if l != s {
		fmt.Println("should be the same logger")
		t.FailNow()
	}
	CustomLogger("github.com/apache/incubator-servicecomb-service-center/pkg/util", "testPkgPath")
	l = Logger()
	if l == LOGGER || l == nil {
		fmt.Println("should create a new instance for 'util'")
		t.FailNow()
	}
	// l.Infof("OK")
}

func BenchmarkLogger(b *testing.B) {
	l := Logger()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Infof("test")
		}
	})
	b.ReportAllocs()
}

func BenchmarkLoggerCustom(b *testing.B) {
	CustomLogger("BenchmarkLoggerCustom", "bmLogger")
	l := Logger()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Infof("test")
		}
	})
	b.ReportAllocs()
}
