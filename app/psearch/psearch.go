package psearch

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

type ChunkInfo struct {
	cBytes       []byte
	lineNoMatch  []int
	lineMatched  []string
	linesInChunk int
	goRoutineNo  int
	seqNo        int
	processed    bool
	searchStr    string
}

func SearchBuffer(ci *ChunkInfo) *ChunkInfo {
	// this method searches the buffer for the passed string
	// params -
	// buf --> the input buffer
	// searchStr --> the string to be searched
	// goRoutineNo --> the goroutine number which is currently processing the buffer
	// seqNo --> its the seq number of the chunk which is being processed
	//ci := ChunkInfo{goRoutineNo: goRoutineNo, seqNo: seqNo}
	// search the buffer for string
	v := bytes.Split(ci.cBytes, []byte("\n"))
	for idx, val := range v {
		ci.linesInChunk += 1
		fmt.Println("procesing bytes", idx, val)
		if bytes.Contains([]byte(val), []byte(ci.searchStr)) {
			fmt.Println("line", idx+1, "xxxx")
			ci.lineNoMatch = append(ci.lineNoMatch, idx+1)
			ci.lineMatched = append(ci.lineMatched, string(val))
		}
	}
	ci.processed = true
	return ci
}

func InitaliseWorkers(iChArr []chan *ChunkInfo, oChArr []chan ChunkInfo, termChArr []chan os.Signal, n int, manChArr []chan bool) {
	// iChArr := make([]chan int, n)
	// oChArr := make([]chan int, n)
	// for i := 0; i < n; i++ {
	// 	v1 := make(chan ChunkInfo)
	// 	v2 := make(chan ChunkInfo)
	// 	v3 := make(chan os.Signal)
	// 	v4 := make(chan bool)
	// 	iChArr = append(iChArr, v1)
	// 	oChArr = append(oChArr, v2)
	// 	termChArr = append(termChArr, v3)
	// 	manChArr = append(manChArr, v4)
	// }

	// spawn n goroutines , each goroutine will accept bytes in the input channel
	// each goroutine will search for a "string" in the incoming bytes and
	// if a there is a match, each goroutine will write the match to output channel

	for i := 0; i < n; i++ {

		go func(iCh chan *ChunkInfo, oCh chan ChunkInfo, termCh chan os.Signal, x int, manCh chan bool) {
			fmt.Println("running goroutine - ", x)
			for {
				select {
				case ci := <-iCh:
					fmt.Println("ich called")
					SearchBuffer(ci)
				case <-termCh:
					close(iCh)
					close(oCh)
					fmt.Println("closing ch -", x)
					return
				case <-manCh:
					close(iCh)
					close(oCh)
					fmt.Println("closing ch -------------", x)
					return

				default:
					fmt.Println("sleeping in ", x)
					time.Sleep(1 * time.Millisecond)
				}
			}
		}(iChArr[i], oChArr[i], termChArr[i], i, manChArr[i])
	}
}

func collectOutputResults(ciArr []*ChunkInfo) {
	// this method collects output from all the goroutine and then
	// do an order by the line number ( of the original file)
	currLine := 0
	fOut := make(map[string]string)
	// loop over the array which is containing ChunkInfo objects
	// and extract useful info like the line number which has a match and the matched line
	for _, ci := range ciArr {
		if !ci.processed {
			// if the chuck is still under processed state then wait for it to get processed
			time.Sleep(5 * time.Second)
		}
		fmt.Println("line number matched ====>", ci.lineNoMatch, ci.seqNo)
		fmt.Println("line matched ====>,", ci.lineMatched)

		// if there is a match found in the current chunk then
		// get the line number of the match
		// line number is a global property and it needs to be considered across the file
		if len(ci.lineNoMatch) > 0 {
			for idx2, val := range ci.lineNoMatch {
				fmt.Println("here", ci.lineNoMatch)
				globalLineNo := strconv.Itoa(val + currLine)
				line := ci.lineMatched[idx2]
				fOut[globalLineNo] = line

			}
		}

		currLine += ci.linesInChunk
		fmt.Println("here2", currLine, ci.linesInChunk)

	}

	fmt.Println("output ===================>", fOut)

}

func MultiRead(filePath string, searchStr string) error {
	var seqChunk []*ChunkInfo
	// check if file exists
	_, err := os.Stat(filePath)
	if err != nil {
		fmt.Println("err")
		return err
	}

	f, err := os.Open(filePath)
	if err != nil {
		// toDO log err
		return err
	}
	defer f.Close()

	// initialise goroutines to handle processing
	n := 10
	iChArr := []chan *ChunkInfo{}
	oChArr := []chan ChunkInfo{}
	termChArr := []chan os.Signal{}
	manChArr := []chan bool{}

	////
	for i := 0; i < n; i++ {
		v1 := make(chan *ChunkInfo)
		v2 := make(chan ChunkInfo)
		v3 := make(chan os.Signal)
		v4 := make(chan bool)
		iChArr = append(iChArr, v1)
		oChArr = append(oChArr, v2)
		termChArr = append(termChArr, v3)
		manChArr = append(manChArr, v4)
	}

	////
	// 	c := make(chan os.Signal, 1)
	// signal.Notify(c, os.Interrupt)
	InitaliseWorkers(iChArr, oChArr, termChArr, n, manChArr)

	// read file in chunks and pass each chunk to the goroutine
	buf := make([]byte, 0, 50)
	reader := bufio.NewReader(f)
	goRoutineNo := 0
	for {
		fmt.Println("loop")
		if goRoutineNo == n {
			goRoutineNo = 1
		}
		n, err1 := reader.Read(buf[:cap(buf)])
		if err1 != nil && err1 != io.EOF {
			fmt.Println(err)
			panic(err1)
		}
		if err1 == io.EOF {
			fmt.Println("reached end")
			for i := 0; i < n; i++ {
				//v := oChArr[i]
				manChArr[i] <- true
				//fmt.Println(v)
			}
			collectOutputResults(seqChunk)
			return nil
		}
		buf = buf[:n]
		ci := ChunkInfo{cBytes: buf, searchStr: searchStr}

		iChArr[goRoutineNo] <- &ci
		// add this sequence to the seqChunk array
		// this array will be used later to read output in a sequence
		seqChunk = append(seqChunk, &ci)
		goRoutineNo += 1
	}

	// for i := 0; i < n; i++ {
	// 	v := oChArr[i]
	// 	fmt.Println(v)
	// }

	return nil

}
