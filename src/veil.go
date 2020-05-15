/*
 * Copyright © 2020 Anurag Dulapalli
 *
 * This library is free software; you can redistribute it and/or
 * modify it under the terms of the GNU Lesser General Public
 * License as published by the Free Software Foundation; either
 * version 2.1 of the License, or (at your option) any later version.
 *
 * This library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public
 * License along with this library; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA
 */

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const accessRuleStringDelimiter string = "~"

// createUnixSocketHTTPClient : Returns a handle to a function that can field and
// filter incoming requests
func createUnixSocketHTTPClient(unixSocketPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", unixSocketPath)
			},
		},
	}
}

// obtainRequestHandler : Returns a handle to a function that can field and
// filter incoming requests
func obtainRequestHandler(targetSocketPath string, accessRulesMap map[string][]string) func(w http.ResponseWriter, r *http.Request) {
	const unauthorizedMsgString string = "{\"type\":\"error\",\"status-code\":401,\"status\":\"Unauthorized\",\"result\":{\"message\":\"access dEEEEEenied\",\"kind\":\"login-required\"}}"
	const unknownMsgString string = "{\"type\":\"error\",\"status-code\":404,\"status\":\"Not Found\",\"result\":{\"message\":\"not found\"}}"
	const badRequestString string = "{\"type\":\"error\",\"status-code\":400,\"status\":\"Invalid Request\",\"result\":{\"message\":\"bad request\"}}"
	const requestTimeoutString string = "{\"type\":\"error\",\"status-code\":408,\"status\":\"Request Timeout\",\"result\":{\"message\":\"request timed out\"}}"
	const internalErrorString string = "{\"type\":\"error\",\"status-code\":500,\"status\":\"Internal Server Error\",\"result\":{\"message\":\"internal server error\"}}"

	var socketHTTPClientPtr *http.Client = createUnixSocketHTTPClient(targetSocketPath)

	// Fields and filters incoming requests, then relays those as
	// appopriate to the encapsulated UNIX Domain Socket
	return func(w http.ResponseWriter, r *http.Request) {
		var requestPath string = "http://unix" + r.URL.Path
		requestContext, _ := context.WithTimeout(context.Background(), 5*time.Second)

		switch r.Method {
		case http.MethodGet:
			fallthrough
		case http.MethodPost:
			fallthrough
		case http.MethodDelete:
			fallthrough
		case http.MethodPatch:
			fallthrough
		case http.MethodPut:
			accessRulesForMethodType, accessRulesExist := accessRulesMap[r.Method]
			if !accessRulesExist {
				io.WriteString(w, unauthorizedMsgString)
				return
			}

			requestAllowanceRuleIndex := sort.SearchStrings(accessRulesForMethodType, r.URL.Path)

			// NOTE: Golang Binary Search returns the length of the slice if
			// the desired element cannot be found.
			if requestAllowanceRuleIndex == len(accessRulesForMethodType) ||
				accessRulesForMethodType[requestAllowanceRuleIndex] != r.URL.Path {
				io.WriteString(w, unauthorizedMsgString)
				return
			}

			httpRequest, errReqCreate := http.NewRequest(r.Method, requestPath, r.Body)
			if errReqCreate != nil {
				io.WriteString(w, internalErrorString)
				return
			}

			httpRequest = httpRequest.WithContext(requestContext)
			response, errReqPeform := (*socketHTTPClientPtr).Do(httpRequest)

			if errReqPeform != nil {
				io.WriteString(w, requestTimeoutString)
				return
			}

			io.Copy(w, response.Body)
			break
		default:
			io.WriteString(w, badRequestString)
		}
	}
}

func createUnixSocketListener(socketPath string) net.Listener {
	os.RemoveAll(socketPath)

	unixListener, err := net.Listen("unix", socketPath)
	if err != nil {
		panic(err)
	}

	return unixListener
}

// readFileLines : Read the contents of a file, and using newlines as the
// delimiter, return a list where each element corresponds with a line from the
// original file.
func readFileLines(filepath string) []string {
	var fileLines []string = []string{}
	if len(filepath) == 0 {
		return fileLines
	}

	file, err := os.Open(filepath)
	if err != nil {
		log.Println("Error opening filepath: ", filepath, err)
		return fileLines
	}

	defer file.Close()

	var scanner *bufio.Scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		currentText := scanner.Text()
		if len(currentText) > 0 {
			fileLines = append(fileLines, scanner.Text())
		}
	}

	if err := scanner.Err(); err != nil {
		log.Println("Error scanning file: ", filepath, err)
	}

	return fileLines
}

// determineAccessRules : Computes a key-value map that describes what HTTP
// requests will be made accessible. Each element in the mapping is from an HTTP
// method type to a list of resource paths.
func determineAccessRules(accessRulesList []string) map[string][]string {
	var accessRulesMap = make(map[string][]string)

	for _, rule := range accessRulesList {
		splitRule := strings.Split(rule, accessRuleStringDelimiter)
		if len(splitRule) != 2 {
			continue
		}

		var ruleHTTPMethod string = splitRule[0]
		var ruleResourcePath string = splitRule[1]
		_, exists := accessRulesMap[ruleHTTPMethod]
		if !exists {
			accessRulesMap[ruleHTTPMethod] = []string{}
		}

		var accessRulesForMethod []string = accessRulesMap[ruleHTTPMethod]
		accessRulesMap[ruleHTTPMethod] = append(accessRulesForMethod, ruleResourcePath)
	}

	for accessRulesMethod := range accessRulesMap {
		accessRulesList := accessRulesMap[accessRulesMethod]
		sort.Strings(accessRulesList)
		accessRulesMap[accessRulesMethod] = accessRulesList
	}

	return accessRulesMap
}

func main() {
	var help *bool = flag.Bool("h", false, "usage help")
	flag.Parse()

	if *help || len(flag.Args()) != 3 {
		fmt.Fprintln(os.Stderr, "usage:", os.Args[0], "<path-to-target-socket> <path-to-exposed-socket> <path-to-access-rules-list>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	var targetSocketPath string = flag.Arg(0)
	var exposedSocketPath string = flag.Arg(1)
	var accessRulesFilepath string = flag.Arg(2)

	log.Println("Launching Unix Socket HTTP Server...")

	var apiAccessHTTPServer http.Server
	apiAccessHTTPServer = http.Server{
		Handler: http.HandlerFunc(obtainRequestHandler(targetSocketPath, determineAccessRules(readFileLines(accessRulesFilepath)))),
	}

	apiAccessHTTPServer.Serve(createUnixSocketListener(exposedSocketPath))
}
