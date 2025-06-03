 
import numpy as np 
import scipy.interpolate as sc
import numpy as np
from scipy import interpolate
import pylab as pl  
import csv
import json


if __name__ == "__main__":
    csv_date = []
    csv_close = []
    csv_day = []
    num = 0   
    # csv file name 
    nameStock = input("input csv: ")
    stockName = input("stock name: ")
    stockTicker = input("stock ticker: ")
    levelNumber = input("level number: ")
    
    with open(nameStock, newline='') as csvfile:
        stockHistory = csv.reader(csvfile, delimiter=',', quotechar='|')

        next(stockHistory, None) # skip the headers
        for row in stockHistory:                                 
            try:   
                x = row[4]
                x = float(x.replace('"', ''))          
                csv_close.append(x)                        
                csv_day.append(num)   
                num = num + 1                  
                csv_date.append(row[0])    
            except ValueError:
                pass
                print("Oops!  That was no valid number. Try again...")

    xnew=np.linspace(0,len(csv_day)-1, len(csv_day))
    pl.plot(csv_day,csv_close,"ro")

    
    # для вида в ["ближайший", "ноль", "линейный", "квадратичный", "кубический"]: # Интерполяция
    # "Ближайший", "ноль" - лестничная интерполяция
    #slinear Линейная интерполяция
    # "quadratic", "cubic" - это интерполяция B-сплайновой кривой 2-го и 3-го порядка
    # ‘slinear’, ‘quadratic’ and ‘cubic’ refer to a spline interpolation of first, second or third order)  
   
    kind = "cubic"
    fInterp1d=interpolate.interp1d(csv_day,csv_close,kind=kind) 
 
    ynew=fInterp1d(xnew)
    pl.plot(xnew,ynew,label=str(kind))
    pl.legend(loc="lower right")
    pl.show()

    # open the file in the write mode
    chartFileName = 'chart_' + stockTicker + '.csv'
    f = open(chartFileName, 'w', newline='')
    # create the csv writer
    writer = csv.writer(f)
    # write a row to the csv file  

    print('len(xnew), ' , len(xnew))
   
    count = 0
    for i in range(50):   
        row = count, fInterp1d(0)
        writer.writerow(row)
        count +=1

    i = 0
    while i < len(xnew)-1:
        row = count, fInterp1d(i)
        writer.writerow(row)
        i += 0.1  
        count += 1
 
    # close the file

    for i in range(50):   
        row = count, ynew[-1]  
        writer.writerow(row)
        count +=1

    f.close()

    # Create a dictionary (which maps to JSON object)
    data = {
    "name": stockName,
    "ticker": stockTicker,
    "chartFile": chartFileName,
    "number": int(levelNumber),
    "finished": False,
    "score": 0
    }

    # Convert to JSON string
    json_string = json.dumps(data, indent=4)
    print(json_string)
    with open(stockTicker + '.json', 'w') as json_file:
        json.dump(data, json_file, indent=4)