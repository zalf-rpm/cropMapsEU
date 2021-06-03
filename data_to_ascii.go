package main

import (
	"bufio"
	"compress/gzip"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/cheggaaa/pb"
	"gonum.org/v1/gonum/stat"
)

const asciiOutFilenameAvg = "avg_%s_trno%s.asc"          // crop_treatmentnumber
const asciiOutFilenameDeviAvg = "devi_avg_%s_trno%s.asc" // crop_treatmentnumber

// USER switch for setting
const USER = "local"

// CROPNAME to analyse
//const CROPNAME = "maize"

// NONEVALUE for ascii table
const NONEVALUE = -9999

// SHOWPROGRESSBAR in cmd line
const SHOWPROGRESSBAR = true

func main() {

	// path to files
	PATHS := map[string]map[string]string{
		"local": {
			"projectdatapath": "./",
			"sourcepath":      "./source/",
			"outputpath":      ".",
			"climate-data":    "./climate-data/corrected/", // path to climate data
			"ascii-out":       "asciigrids_debug/",         // path to ascii grids
			"png-out":         "png_debug/",                // path to png images
			"pdf-out":         "pdf-out_debug/",            // path to pdf package
		},
		"test": {
			"projectdatapath": "./",
			"sourcepath":      "./source/",
			"outputpath":      "./testout/",
			"climate-data":    "./climate-data/corrected/", // path to climate data
			"ascii-out":       "asciigrids2/",              // path to ascii grids
			"png-out":         "png2/",                     // path to png images
			"pdf-out":         "pdf-out2/",                 // path to pdf package
		},
		"Cluster": {
			"projectdatapath": "/project/",
			"sourcepath":      "/source/",
			"outputpath":      "/out/",
			"climate-data":    "/climate-data/", // path to climate data
			"ascii-out":       "asciigrid/",     // path to ascii grids
			"png-out":         "png/",           // path to png images
			"pdf-out":         "pdf-out/",       // path to pdf package
		},
	}

	// command line flags
	pathPtr := flag.String("path", USER, "path id")
	sourcePtr := flag.String("source", "", "path to source folder")
	outPtr := flag.String("out", "", "path to out folder")
	noprogessPtr := flag.Bool("showprogess", SHOWPROGRESSBAR, "show progress bar")
	projectPtr := flag.String("project", "", "path to project folder")
	cropNamePtr := flag.String("crop", "maize", "name of crop to calc")

	flag.Parse()

	pathID := *pathPtr
	showBar := *noprogessPtr
	sourceFolder := *sourcePtr
	outputFolder := *outPtr
	projectpath := *projectPtr
	cropName := *cropNamePtr

	if len(sourceFolder) == 0 {
		sourceFolder = PATHS[pathID]["sourcepath"]
	}
	if len(outputFolder) == 0 {
		outputFolder = PATHS[pathID]["outputpath"]
	}
	if len(projectpath) == 0 {
		projectpath = PATHS[pathID]["projectdatapath"]
	}

	asciiOutFolder := filepath.Join(outputFolder, PATHS[pathID]["ascii-out"])

	gridSource := filepath.Join(projectpath, "stu_eu_layer_grid.csv")
	extRow, extCol, gridSourceLookup := GetGridLookup(gridSource)

	filelist, err := ioutil.ReadDir(sourceFolder)
	if err != nil {
		log.Fatal(err)
	}
	maxRefNo := len(filelist) // size of the list
	for _, file := range filelist {
		refIDStr := strings.Split(strings.Split(file.Name(), ".")[0], "_")[3]
		refID64, err := strconv.ParseInt(refIDStr, 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		if maxRefNo < int(refID64) {
			maxRefNo = int(refID64)
		}
	}

	numInput := len(filelist)
	var p ProcessedData
	p.maxAllAvgYield = 0.0
	p.maxSdtDeviation = 0.0
	p.allGrids = make(map[SimKeyTuple][]int)
	p.StdDevAvgGrids = make(map[SimKeyTuple][]int)
	p.outputGridsGenerated = false
	p.currentInput = 0
	p.progress = progress(numInput, "input files")

	outChan := make(chan bool)

	currRuns := 0
	maxRuns := 60
	// iterate over all model run results
	for _, sourcefileInfo := range filelist {

		go func(sourcefileName string, outC chan bool) {
			//sourcefileName := sourcefileInfo.Name()
			sourcefile, err := os.Open(filepath.Join(sourceFolder, sourcefileName))
			if err != nil {
				log.Fatal(err)
			}
			defer sourcefile.Close()
			refIDStr := strings.Split(strings.Split(sourcefileName, ".")[0], "_")[3]
			refID64, err := strconv.ParseInt(refIDStr, 10, 64)
			if err != nil {
				log.Fatal(err)
			}
			refIDIndex := int(refID64) - 1
			simulations := make(map[SimKeyTuple][]float64)
			dateYearOrder := make(map[SimKeyTuple][]int)

			firstLine := true
			var header SimDataIndex
			scanner := bufio.NewScanner(sourcefile)
			for scanner.Scan() {
				line := scanner.Text()
				if firstLine {
					// read header
					firstLine = false
					header = readHeader(line)
				} else {
					// load relevant line content
					lineKey, lineContent, lineErr := loadLine(line, header)
					if lineErr != nil {
						log.Printf("%v :%s", lineErr, sourcefileName)
						break
					}
					// check for the lines with a specific crop
					if IsCrop(lineKey, cropName) && (lineKey.treatNo == "T1" || lineKey.treatNo == "T2") {
						yieldValue := lineContent.yields
						yearValue := lineContent.year
						if _, ok := simulations[lineKey]; !ok {
							simulations[lineKey] = make([]float64, 0, 30)
							dateYearOrder[lineKey] = make([]int, 0, 30)
						}

						simulations[lineKey] = append(simulations[lineKey], yieldValue)
						dateYearOrder[lineKey] = append(dateYearOrder[lineKey], yearValue)
					}
				}
			}

			p.setOutputGridsGenerated(simulations, maxRefNo)

			for simKey := range simulations {
				pixelValue := CalculatePixel(simulations[simKey])
				p.setMaxAllAvgYield(pixelValue)
				stdDeviation := stat.StdDev(simulations[simKey], nil)
				p.setMaxSdtDeviation(stdDeviation)

				p.allGrids[simKey][refIDIndex] = int(pixelValue)
				p.StdDevAvgGrids[simKey][refIDIndex] = int(stdDeviation)
			}

			p.incProgressBar(showBar)
			outChan <- true

		}(sourcefileInfo.Name(), outChan)
		currRuns++
		if currRuns >= maxRuns {
			for currRuns >= maxRuns {
				<-outChan
				currRuns--
				// select {
				// case <-outChan:
				// 	currRuns--
				// }
			}
		}
	}
	for currRuns > 0 {
		<-outChan
		currRuns--
		// select {
		// case <-outChan:
		// 	currRuns--
		// }
	}

	outC := make(chan string)
	waitForNum := 1

	go drawDateMaps(gridSourceLookup,
		p.allGrids,
		asciiOutFilenameAvg,
		extCol, extRow,
		asciiOutFolder,
		"Average Yield - Scn: %v %v %v",
		"Yield in t",
		false,
		"jet",
		nil, nil, 0.001, NONEVALUE,
		int(p.maxAllAvgYield),
		"average yield grids", outC)

	waitForNum++
	go drawDateMaps(gridSourceLookup,
		p.StdDevAvgGrids,
		asciiOutFilenameDeviAvg,
		extCol, extRow,
		asciiOutFolder,
		"Std Deviation - Scn: %v %v %v",
		"standart deviation",
		false,
		"cool",
		nil, nil, 1.0, 0,
		int(p.maxSdtDeviation),
		"std average yield grids", outC)

	for waitForNum > 0 {
		progessStatus := <-outC
		waitForNum--
		fmt.Println(progessStatus)
		// select {
		// case progessStatus := <-outC:
		// 	waitForNum--
		// 	fmt.Println(progessStatus)
		// }
	}

}

// SimKeyTuple key to identify each simulatio setup
type SimKeyTuple struct {
	treatNo        string
	climateSenario string
	crop           string
	comment        string
}

// SimData simulation data from a line
type SimData struct {
	year   int
	yields float64
}

// GridCoord tuple of positions
type GridCoord struct {
	row int
	col int
}

// SimDataIndex indices for climate data
type SimDataIndex struct {
	treatNoIdx        int
	climateSenarioIdx int
	cropIdx           int
	commentIdx        int
	yearIdx           int
	yieldsIdx         int
}

// ProcessedData combined data from results
type ProcessedData struct {
	maxAllAvgYield       float64
	maxSdtDeviation      float64
	allGrids             map[SimKeyTuple][]int
	StdDevAvgGrids       map[SimKeyTuple][]int
	outputGridsGenerated bool
	mux                  sync.Mutex
	currentInput         int
	progress             progressfunc
}

func (p *ProcessedData) setOutputGridsGenerated(simulations map[SimKeyTuple][]float64, maxRefNo int) bool {

	p.mux.Lock()
	out := false
	if !p.outputGridsGenerated {
		p.outputGridsGenerated = true
		out = true
		for simKey := range simulations {
			p.allGrids[simKey] = newGridLookup(maxRefNo, NONEVALUE)
			p.StdDevAvgGrids[simKey] = newGridLookup(maxRefNo, NONEVALUE)
		}
	}
	p.mux.Unlock()
	return out
}

func (p *ProcessedData) setMaxAllAvgYield(pixelValue float64) {
	p.mux.Lock()
	if pixelValue > p.maxAllAvgYield {
		p.maxAllAvgYield = pixelValue
	}
	p.mux.Unlock()
}
func (p *ProcessedData) setMaxSdtDeviation(stdDeviation float64) {
	p.mux.Lock()
	if stdDeviation > p.maxSdtDeviation {
		p.maxSdtDeviation = stdDeviation
	}
	p.mux.Unlock()
}

func isSeperator(r rune) bool {
	return r == ';' || r == ','
}
func readHeader(line string) SimDataIndex {
	//read header
	tokens := strings.FieldsFunc(line, isSeperator)
	indices := SimDataIndex{
		treatNoIdx:        -1,
		climateSenarioIdx: -1,
		cropIdx:           -1,
		commentIdx:        -1,
		yearIdx:           -1,
		yieldsIdx:         -1,
	}

	for i, token := range tokens {
		t := strings.Trim(token, "\"")
		switch t {
		case "Crop":
			indices.cropIdx = i
		case "sce":
			indices.climateSenarioIdx = i
		case "Yield":
			indices.yieldsIdx = i
		case "ProductionCase":
			indices.commentIdx = i
		case "TrtNo":
			indices.treatNoIdx = i
		case "TrNo":
			indices.treatNoIdx = i
		case "Year":
			indices.yearIdx = i
		}
	}
	return indices
}

func loadLine(line string, header SimDataIndex) (SimKeyTuple, SimData, error) {
	// read relevant content from line
	rawTokens := strings.FieldsFunc(line, isSeperator)

	tokens := make([]string, len(rawTokens))
	for i, token := range rawTokens {
		tokens[i] = strings.Trim(token, "\"")
	}

	var key SimKeyTuple
	var content SimData
	key.treatNo = tokens[header.treatNoIdx]
	key.climateSenario = tokens[header.climateSenarioIdx]
	key.crop = tokens[header.cropIdx]
	key.comment = tokens[header.commentIdx]
	val, err := strconv.ParseInt(tokens[header.yearIdx], 10, 0)
	if err != nil {
		return key, content, err
	}
	content.year = int(val)
	content.yields, _ = strconv.ParseFloat(tokens[header.yieldsIdx], 64)
	return key, content, nil
}

func newGridLookup(maxRef, defaultVal int) []int {
	grid := make([]int, maxRef)
	for i := 0; i < maxRef; i++ {
		grid[i] = defaultVal
	}
	return grid
}

// GetGridLookup ..
func GetGridLookup(gridsource string) (rowExt int, colExt int, lookupGrid [][]int) {
	colExt = 0
	rowExt = 0
	lookup := make(map[int64][]GridCoord)

	sourcefile, err := os.Open(gridsource)
	if err != nil {
		log.Fatal(err)
	}
	defer sourcefile.Close()
	firstLine := true
	colID := -1
	rowID := -1
	refID := -1
	scanner := bufio.NewScanner(sourcefile)
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Split(line, ",")
		if firstLine {
			firstLine = false
			for index, token := range tokens {
				if token == "Column_" {
					colID = index
				}
				if token == "Row" {
					rowID = index
				}
				if token == "soil_ref" {
					refID = index
				}
			}
		} else {
			col, _ := strconv.ParseInt(tokens[colID], 10, 64)
			row, _ := strconv.ParseInt(tokens[rowID], 10, 64)
			ref, _ := strconv.ParseInt(tokens[refID], 10, 64)
			if int(col) > colExt {
				colExt = int(col)
			}
			if int(row) > rowExt {
				rowExt = int(row)
			}
			if _, ok := lookup[ref]; !ok {
				lookup[ref] = make([]GridCoord, 0, 1)
			}
			lookup[ref] = append(lookup[ref], GridCoord{int(row), int(col)})
		}
	}
	lookupGrid = newGrid(rowExt, colExt, NONEVALUE)
	for ref, coord := range lookup {
		for _, rowCol := range coord {
			lookupGrid[rowCol.row-1][rowCol.col-1] = int(ref)
		}
	}

	return rowExt, colExt, lookupGrid
}

func newGrid(extRow, extCol, defaultVal int) [][]int {
	grid := make([][]int, extRow)
	for r := 0; r < extRow; r++ {
		grid[r] = make([]int, extCol)
		for c := 0; c < extCol; c++ {
			grid[r][c] = defaultVal
		}
	}
	return grid
}

// IsCrop ...
func IsCrop(key SimKeyTuple, cropName string) bool {
	return strings.HasPrefix(key.crop, cropName)
}
func average(list []float64) float64 {
	sum := 0.0
	val := 0.0
	lenVal := 0.0
	for _, x := range list {
		if x >= 0 {
			sum = sum + x
			lenVal++
		}
	}
	if lenVal > 0 {
		val = sum / lenVal
	}

	return val
}

// CalculatePixel yield average for stable yield set
func CalculatePixel(yieldList []float64) float64 {
	pixelValue := average(yieldList)
	return pixelValue
}

func makeDir(outPath string) {
	dir := filepath.Dir(outPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			log.Fatalf("ERROR: Failed to generate output path %s :%v", dir, err)
		}
	}
}

type progressfunc func(int)

func progress(total int, status string) func(int) {
	count := total
	current := 0
	bar := pb.New(count)
	// show percents (by default already true)
	bar.ShowPercent = true
	//show bar (by default already true)
	bar.ShowBar = true
	bar.ShowCounters = true
	bar.ShowTimeLeft = true
	bar.Start()
	return func(newCurrent int) {
		if newCurrent > current {
			inc := newCurrent - current

			for i := 0; i < inc && current < count; i++ {
				current++
				if current == count {
					bar.FinishPrint("The End!")
				}
				bar.Increment()
			}
		}
	}
}
func (p *ProcessedData) incProgressBar(showBar bool) {
	p.mux.Lock()
	p.currentInput++
	if showBar {
		p.progress(p.currentInput)
	}
	p.mux.Unlock()
}

func drawDateMaps(gridSourceLookup [][]int, grids map[SimKeyTuple][]int, filenameFormat string, extCol, extRow int, asciiOutFolder, titleFormat, labelText string, showBar bool, colormap string, cbarLabel []string, ticklist []float64, factor float64, minVal, maxVal int, progessStatus string, outC chan string) {

	var currentInput int
	var numInput int
	var progressBar func(int)
	if showBar {
		numInput = len(grids)
		progressBar = progress(numInput, progessStatus)
	}

	for simKey, simVal := range grids {
		//simkey = treatmentNo, climateSenario, maturityGroup, comment
		gridFileName := fmt.Sprintf(filenameFormat, simKey.crop, simKey.treatNo)
		gridFileName = strings.ReplaceAll(gridFileName, "/", "-") //remove directory seperator from filename
		gridFilePath := filepath.Join(asciiOutFolder, simKey.climateSenario, gridFileName)
		file := writeAGridHeader(gridFilePath, extCol, extRow)

		writeRows(file, extRow, extCol, simVal, gridSourceLookup)
		file.Close()
		title := fmt.Sprintf(titleFormat, simKey.climateSenario, simKey.crop, simKey.comment)
		writeMetaFile(gridFilePath, title, labelText, colormap, nil, cbarLabel, ticklist, factor, maxVal, minVal, "")

		if showBar {
			currentInput++
			progressBar(currentInput)
		}
	}
	outC <- progessStatus
}

func writeAGridHeader(name string, nCol, nRow int) (fout Fout) {
	cornerX := 0.0
	cornery := 0.0
	novalue := -9999
	cellsize := 1.0
	// create an ascii file, which contains the header
	makeDir(name)
	file, err := os.OpenFile(name+".gz", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}

	gfile := gzip.NewWriter(file)
	fwriter := bufio.NewWriter(gfile)
	fout = Fout{file, gfile, fwriter}

	fout.Write(fmt.Sprintf("ncols %d\n", nCol))
	fout.Write(fmt.Sprintf("nrows %d\n", nRow))
	fout.Write(fmt.Sprintf("xllcorner     %f\n", cornerX))
	fout.Write(fmt.Sprintf("yllcorner     %f\n", cornery))
	fout.Write(fmt.Sprintf("cellsize      %f\n", cellsize))
	fout.Write(fmt.Sprintf("NODATA_value  %d\n", novalue))

	return fout
}

func writeRows(fout Fout, extRow, extCol int, simGrid []int, gridSourceLookup [][]int) {
	size := len(simGrid)
	for row := 0; row < extRow; row++ {

		for col := 0; col < extCol; col++ {
			refID := gridSourceLookup[row][col]
			if refID >= 0 && refID < size {
				fout.Write(strconv.Itoa(simGrid[refID-1]))
				fout.Write(" ")
				//line += fmt.Sprintf("%d ", simGrid[refID-1])
			} else {
				fout.Write("-9999 ")
				//line += "-9999 "
			}
		}
		fout.Write("\n")
		//line += "\n"
	}
	//file.WriteString(line)
}

// Fout combined file writer
type Fout struct {
	file    *os.File
	gfile   *gzip.Writer
	fwriter *bufio.Writer
}

// Write string to zip file
func (f Fout) Write(s string) {
	f.fwriter.WriteString(s)
}

// Close file writer
func (f Fout) Close() {
	f.fwriter.Flush()
	// Close the gzip first.
	f.gfile.Close()
	f.file.Close()
}

func writeMetaFile(gridFilePath, title, labeltext, colormap string, colorlist []string, cbarLabel []string, ticklist []float64, factor float64, maxValue, minValue int, minColor string) {
	metaFilePath := gridFilePath + ".meta"
	makeDir(metaFilePath)
	file, err := os.OpenFile(metaFilePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	file.WriteString(fmt.Sprintf("title: '%s'\n", title))
	file.WriteString(fmt.Sprintf("labeltext: '%s'\n", labeltext))
	if colormap != "" {
		file.WriteString(fmt.Sprintf("colormap: '%s'\n", colormap))
	}
	if colorlist != nil {
		file.WriteString("colorlist: \n")
		for _, item := range colorlist {
			file.WriteString(fmt.Sprintf(" - '%s'\n", item))
		}
	}
	if cbarLabel != nil {
		file.WriteString("cbarLabel: \n")
		for _, cbarItem := range cbarLabel {
			file.WriteString(fmt.Sprintf(" - '%s'\n", cbarItem))
		}
	}
	if ticklist != nil {
		file.WriteString("ticklist: \n")
		for _, tick := range ticklist {
			file.WriteString(fmt.Sprintf(" - %f\n", tick))
		}
	}
	file.WriteString(fmt.Sprintf("factor: %f\n", factor))
	if maxValue != NONEVALUE {
		file.WriteString(fmt.Sprintf("maxValue: %d\n", maxValue))
	}
	if minValue != NONEVALUE {
		file.WriteString(fmt.Sprintf("minValue: %d\n", minValue))
	}
	if len(minColor) > 0 {
		file.WriteString(fmt.Sprintf("minColor: %s\n", minColor))
	}
}
