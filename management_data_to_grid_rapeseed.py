#!/usr/bin/python
# -*- coding: UTF-8

from dataclasses import dataclass
import csv
import geopip

def build() :
    "main"

    gridFile = "./stu_eu_layer_grid.csv"
    input = "./rapeseed_dates.csv"
    outFile = "./stu_eu_layer_grid_management_rapeseed.csv"
    
    lookup = readGridLookup(gridFile)

    # create a lat/lon map, assigning each Datapoint a lat/lon value
    # interpolate for each stu_eu grid point the matching data point
    dates = dict()
    with open(input) as _:
        reader = csv.reader(_)
        next(reader)
        for row in reader:
            
            #Country,Start_Sowing,End_Sowing,Start_Harvest,End_Harvest
            country = row[0]
            dates[country] = {
                "sowStart": int(row[1]),
                "sowEnd":   int(row[2]),
                "harvest":  int(row[4]),
            }
        
    notListed = set()
    for key in lookup :
        dateEntry = lookup[key]
        result = geopip.search(lng=dateEntry.lon, lat=dateEntry.lat)
        if result :
            area = result['NAME']
            if area in dates :
                lookup[key].dates["sowStart"] = dates[area]["sowStart"]
                lookup[key].dates["sowEnd"] = dates[area]["sowEnd"]
                lookup[key].dates["harvest"] = dates[area]["harvest"]
            else :
                notListed.add(area)

    print(notListed)

    outGridHeader = "soil_ref,latitude,longitude,sowStart,sowEnd,harvest\n"
    with open(outFile, mode="wt", newline="") as outGridfile :
        outGridfile.writelines(outGridHeader)
        for key in lookup : 
            data = lookup[key]
            outline = [str(data.soilref),
                        str(data.lat), #lat
                        str(data.lon), #long
                        str(data.dates["sowStart"]),
                        str(data.dates["sowEnd"]),
                        str(data.dates["harvest"])
                        ]
            outGridfile.writelines(",".join(outline) + "\n")

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
                    datesDic["sowStart"] = 152
                    datesDic["sowEnd"] = 181
                    datesDic["harvest"] = 335
                    lookup[soilref] = DateEntry(soilref, lat, lon, datesDic)
    return lookup    

if __name__ == "__main__":
    build()