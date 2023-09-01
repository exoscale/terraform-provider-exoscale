package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

type TestCall struct {
	Req        interface{}
	Resp       interface{}
	RespStatus int
}

type TestFlow struct {
	Calls []TestCall
}

var (
	testdataFilename = "testdata.json"
	mu               sync.Mutex
)

func ReadTestdata(fileName string) (*TestFlow, error) {
	tf := &TestFlow{}

	f, err := os.OpenFile(fileName, os.O_RDONLY, 0666)
	if err != nil {
		if os.IsNotExist(err) {
			return tf, nil
		}
		return nil, fmt.Errorf("error opening file for reading %w", err)
	}
	defer f.Close()

	cntnt, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("error reading all test data %w", err)
	}

	err = json.Unmarshal(cntnt, tf)
	if err != nil {
		return &TestFlow{}, nil
	}

	return tf, nil
}

func GetTestCall(callNr int, callResp interface{}) error {
	testflow, err := ReadTestdata(testdataFilename)
	if err != nil {
		return fmt.Errorf("error reading test data %w", err)
	}

	if len(testflow.Calls) <= callNr {
		return fmt.Errorf("no call %d", callNr)
	}

	callIfval := testflow.Calls[callNr]
	callIfvalJson, err := json.Marshal(callIfval)
	if err != nil {
		return fmt.Errorf("error marshalling call %w", err)
	}

	err = json.Unmarshal(callIfvalJson, callResp)
	if err != nil {
		return fmt.Errorf("error unmarshalling call %w", err)
	}

	return nil
}

func WriteTestdata(req, resp interface{}, respStatus int) error {
	mu.Lock()
	defer mu.Unlock()

	tf, err := ReadTestdata(testdataFilename)
	if err != nil {
		return fmt.Errorf("error reading test data before writing %w", err)
	}

	tf.Calls = append(tf.Calls, TestCall{
		RespStatus: respStatus,
		Req:        req,
		Resp:       resp,
	})

	indentedJSON, err := json.MarshalIndent(tf, "", "    ")
	if err != nil {
		return fmt.Errorf("error marshalling with ident %w", err)
	}

	f, err := os.Create(testdataFilename)
	if err != nil {
		return fmt.Errorf("error opening file for writing %w", err)
	}
	defer f.Close()

	_, err = f.Write(indentedJSON)
	if err != nil {
		return fmt.Errorf("error writing testdata %w", err)
	}

	return nil
}
