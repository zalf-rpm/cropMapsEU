#!/usr/bin/python
# -*- coding: UTF-8

import gzip
from dataclasses import dataclass
import os
import numpy as np

def build() :
    "main"

    gridFile = "./stu_eu_layer_grid.csv"
    pathSowingStartM = "./asciigrids_debug/Maize.crop.calendar.fill/plant.start.asc"
    pathSowingEndM = "./asciigrids_debug/Maize.crop.calendar.fill/plant.end.asc"
    pathEndHarvestM = "./asciigrids_debug/Maize.crop.calendar.fill/harvest.end.asc"

    pathSowingStartWW = "./asciigrids_debug/Wheat.Winter.crop.calendar.fill/plant.start.asc"
    pathSowingEndWW = "./asciigrids_debug/Wheat.Winter.crop.calendar.fill/plant.end.asc"
    pathEndHarvestWW = "./asciigrids_debug/Wheat.Winter.crop.calendar.fill/harvest.end.asc"

    outFile = "./stu_eu_layer_grid_management.csv"
    
    lookup = readGridLookup(gridFile)

    # create a lat/lon map, assigning each Datapoint a lat/lon value
    # interpolate for each stu_eu grid point the matching data point

    def loadDates(file, dateKey) :
        header = readAsciiHeader(file)
        ascii_data_array = np.loadtxt(header.ascii_path, dtype=np.float, skiprows=6)
        if ascii_data_array.ndim == 1 :
            reshaped_to_2d = np.reshape(ascii_data_array, (header.ascii_rows,-1 ))
            ascii_data_array = reshaped_to_2d
        ascii_data_array[ascii_data_array == header.ascii_nodata] = -1

        colsLon = header.ascci_cols
        rowsLat = header.ascii_rows
        for key in lookup :
            date = lookup[key]
            # calculate position of lon/lat in array
            arrPosCol = ((colsLon//2) + int(date.lon*12.))
            arrPosRow = ((rowsLat//2) + int(-1*date.lat*12.))
            value = int(ascii_data_array[arrPosRow][arrPosCol])
            if value > 0 and value < 365 :
                lookup[key].dates[dateKey] = value

    loadDates( pathSowingStartM,"sowStartM")
    loadDates(pathSowingStartWW ,"sowStartW")
    loadDates( pathSowingEndM,"sowEndM")
    loadDates( pathSowingEndWW,"sowEndW")
    loadDates( pathEndHarvestM,"harvestM")
    loadDates(pathEndHarvestWW ,"harvestW")

    counter1 = 0
    counter2 = 0
    for key in lookup :
        if lookup[key].dates["harvestM"] > lookup[key].dates["sowStartW"] :
            counter1 += 1
            lookup[key].dates["harvestM"] = 257
            if lookup[key].dates["sowStartW"] < 258 :
                lookup[key].dates["sowStartW"] = 258
 
            if lookup[key].dates["sowEndW"] < lookup[key].dates["sowStartW"] :
                counter2 += 1
                lookup[key].dates["sowEndW"] = lookup[key].dates["sowStartW"]+14

    print(counter1, counter2)

    outGridHeader = "soil_ref,latitude,longitude,sowStartM,sowStartW,sowEndM,sowEndW,harvestM,harvestW \n"
    with open(outFile, mode="wt", newline="") as outGridfile :
        outGridfile.writelines(outGridHeader)
        for key in lookup : 
            data = lookup[key]
            outline = [str(data.soilref),
                        str(data.lat), #lat
                        str(data.lon), #long
                        str(data.dates["sowStartM"]),
                        str(data.dates["sowStartW"]),
                        str(data.dates["sowEndM"]),
                        str(data.dates["sowEndW"]),
                        str(data.dates["harvestM"]),
                        str(data.dates["harvestW"])
                        ]
            outGridfile.writelines(",".join(outline) + "\n")
@dataclass
class AsciiHeader:
    ascii_path: str
    ascci_cols: int
    ascii_rows: int
    ascii_xll: float
    ascii_yll: float
    ascii_cs: float
    ascii_nodata: float
    image_extent: list

def readAsciiHeader(ascii_path) :
    if ascii_path.endswith(".gz") :
           # Read in ascii header data
        with gzip.open(ascii_path, 'rt') as source:
            ascii_header = source.readlines()[:6] 
    else :
        # Read in ascii header data
        with open(ascii_path, 'r') as source:
            ascii_header = source.readlines()[:6]

    # Read the ASCII raster header
    ascii_header = [item.strip().split()[-1] for item in ascii_header]
    ascci_cols = int(ascii_header[0])
    ascii_rows = int(ascii_header[1])
    ascii_xll = float(ascii_header[2])
    ascii_yll = float(ascii_header[3])
    ascii_cs = float(ascii_header[4])
    ascii_nodata = float(ascii_header[5])

    image_extent = [
                ascii_xll, ascii_xll + ascci_cols * ascii_cs,
                ascii_yll, ascii_yll + ascii_rows * ascii_cs] 

    return AsciiHeader(ascii_path, ascci_cols, ascii_rows, ascii_xll, ascii_yll, ascii_cs, ascii_nodata, image_extent)

@dataclass
class DateEntry:
    soilref: int
    lat: float  
    lon: float
    dates: dict


def readGridLookup(gridsource) :
    lookup = dict()    
    with open(gridsource) as sourcefile:
        firstLine = True
        soilrefID = -1
        latID = -1
        lonID = -1
        for line in sourcefile:
            tokens = line.strip().split(",")          
            if firstLine :
                # read header
                firstLine = False
                for index in range(len(tokens)) :
                    token = tokens[index]
                    if token == "latitude" :
                        latID = index
                    if token == "longitude" :
                        lonID = index
                    if token == "soil_ref" :
                        soilrefID = index
            else :
                soilref = int(tokens[soilrefID])
                lat = float(tokens[latID])
                lon = float(tokens[lonID])
                if not (soilref in lookup) :
                    datesDic = dict()
                    datesDic["sowStartM"] = 74
                    datesDic["sowStartW"] = 288
                    datesDic["sowEndM"] = 135
                    datesDic["sowEndW"] = 319
                    datesDic["harvestM"] = 287
                    datesDic["harvestW"] = 248
                    lookup[soilref] = DateEntry(soilref, lat, lon, datesDic)
    return lookup    

if __name__ == "__main__":
    build()