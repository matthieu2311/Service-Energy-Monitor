import matplotlib.pyplot as plt
import json
import datetime as dt
import requests
import numpy as np
import math



holes = {
        "~TIME_MIN_CONS~":"error", "~VALUE_MIN_CONS~":"error", "~TIME_MAX_CONS~":"error", 
        "~VALUE_MAX_CONS~":"error", "~SUM_CONS~":"error", "~GRAPH_IMAGE_SRC~":"error", "~SERVER_URL~":"error",
        "~BAR_IMAGE_SRC~":"error", "~USER_ID~":"error", "~PIE_CHART_IMG_SRC~":"error", "~PHONE_CHARGES_EQ~":"error",
        "~TV_HOURS_EQ~":"error","~KM_EQ~":"error","~PRICE_EQ~":"error",
        "~DAY_RANK~":"error", "~WEEK_RANK~":"error", "~MONTH_RANK~":"error", "~YEAR_RANK~":"error", "~NBR_USERS~":"error", 
        "~DAY_STRING~":"error", "~WEEK_STRING~":"error", "~MONTH_STRING~":"error", "~YEAR_STRING~":"error"
    }

auth = ('','') # Put your username and password in the auth variable, that will then be passed for each of the get requests. 
               # Useful for accessing grid5000 reverse proxy for example, but not necessary for a local server. 


def convertJSONToDict(jsonFile):
    with open(jsonFile, 'r') as file:
        return json.load(file)



def displayClassicGraph(url, id):

    imageURL = url + "users/"+str(id)+"/consumption"

    response = requests.get(imageURL, auth=auth, verify=False) 

    if response.status_code == 200:
        print("GET request successful, now procceding to create the image")
    else:
        print("Something went wrong")
        response.raise_for_status()
    dict = response.json()
    
    temp = []
    
    for i in range(len(dict)):
        elt = dict[i]
        temp.append((elt["timestamp"], elt["value"]))
            
    
    
    temp.sort()
    x, y = [], []
    
    for i in range(0, len(temp)):   # timestamp format -> 2025-02-10T14:49:30Z 
        a = temp[i]
        year = a[0][0:4]
        month = a[0][5:7]
        day = a[0][8:10]
        hour = a[0][11:13]
        minute = a[0][14:16]
        sec = a[0][17:-1]
    
        x.append(dt.datetime(int(year), int(month), int(day), int(hour), int(minute), int(sec)))
        y.append(a[1])
    
    fig, ax = plt.subplots()
    ax.bar(x, y)
    plt.ylabel("mWh")
    #If using linux, the data is in J : plt.ylabel("J")
    plt.legend(["Mean consumption value per day"])
    plt.xticks(rotation=45)
    plt.savefig(__file__.replace("client.py", "").replace("\\", "/")+"images/u"+str(id)+"Consumption.png") 
    # To make the title of the image unique and to store all images created,
    #  we can use the time of creation in the title : +str(dt.datetime.today())[12:].replace(":", "").replace(".", "")+".png")
    plt.show()
    holes["~GRAPH_IMAGE_SRC~"] = "./images/u"+str(id)+"Consumption.png"
    holes["~USER_ID~"] = str(id)



def displayTodayBarChart(url, id):

    todayURL = url + "users/"+str(id)+"/today"
    response = requests.get(todayURL, auth=auth, verify=False)

    if response.status_code == 200:
        print("GET request successful")
    else:
        print("Something went wrong")
        response.raise_for_status()

    dict = response.json()
    print(dict)

    holes["~TIME_MAX_CONS~"] = dict[0]["timestamp"].replace('T', ' ').replace('Z', '')
    holes["~TIME_MIN_CONS~"] = dict[1]["timestamp"].replace('T', ' ').replace('Z', '')
    holes["~SUM_CONS~"] = round(dict[2]["value"],2)
    holes["~VALUE_MAX_CONS~"] = round(dict[0]["value"], 2) 
    holes["~VALUE_MIN_CONS~"] = round(dict[1]["value"], 2)
    holes["~SERVER_URL~"] = url

    # Formulas with data in mWh (DEMETER/csv version)
    #holes["~PHONE_CHARGES_EQ~"] = round(holes["~SUM_CONS~"]*(10**-3)/15, 2) #Assuming 15Wh for one charge
    #holes["~TV_HOURS_EQ~"] = round(holes["~SUM_CONS~"]*10**-6*4) #Assuming 1kWh = 4h of TV
    #holes["~KM_EQ~"] = round(holes["~SUM_CONS~"] * 10**-6*2) #Assuming 1kWh = 2km with an electric smart 
    #holes["~PRICE_EQ~"] = round(holes["~SUM_CONS~"] * 10**-6 * 0.2016, 3) #Price of 1kWh from EDF in february 2025

    #Formulas with data in J (Linux version)
    holes["~PHONE_CHARGES_EQ~"] = round(holes["~SUM_CONS~"]/3.6*(10**-3)/15, 2) 
    holes["~TV_HOURS_EQ~"] = round(holes["~SUM_CONS~"]/3.6*10**-6*4) 
    holes["~KM_EQ~"] = round(holes["~SUM_CONS~"]/3.6 * 10**-6*2) 
    holes["~PRICE_EQ~"] = round(holes["~SUM_CONS~"]/3.6 * 10**-6 * 0.2016, 3) 




    height = [holes["~SUM_CONS~"]]
    bars = ("sum",)
    y_pos = np.arange(len(bars))
    plt.bar(y_pos, height, color=["lightblue"])
    plt.xticks(y_pos, bars)
    #In J for linux, in mWh for DEMETER/csv
    plt.ylabel("J")
    plt.legend(["Today's total consumption"])
    
    plt.savefig(__file__.replace("client.py", "").replace("\\", "/")+"images/u"+str(id)+"BarChart.png")
    plt.show()
    holes["~BAR_IMAGE_SRC~"] = "./images/u"+str(id)+"BarChart.png"


def displayWeeklyBarChart(url, id):

    weeklyURL = url + "users/"+str(id)+"/weeklyMean"
    response = requests.get(weeklyURL, auth=auth, verify=False)

    if response.status_code == 200:
        print("GET request successful")
    else:
        print("Something went wrong")
        response.raise_for_status()

    dict = response.json()


    heights = [dict[k] for k in range(len(dict))]
    y_pos = np.arange(len(heights))
    plt.bar(y_pos, heights[::-1])
    plt.xticks(y_pos)
    plt.savefig(__file__.replace("client.py", "").replace("\\", "/")+"images/u"+str(id)+"WeeklyBarChart.png")
    #In J for linux, in mWh for DEMETER/csv
    plt.ylabel("J")
    plt.legend(["Weekly mean consumption"])
    plt.show()
    holes["~WEEKLY_BAR_IMAGE_SRC~"] = "./images/u"+str(id)+"WeeklyBarChart.png"


def displayRankings(url, id):
    ranksURL = url + "users/"+str(id)+"/rank"
    response = requests.get(ranksURL, auth=auth, verify=False)

    if response.status_code == 200:
        print("GET request successful")
    else:
        print("Something went wrong")
        response.raise_for_status()

    dict = response.json()

    #Categories : top 1%, top 5%, top 10%, top 20%, top 50%.
    onePercent = math.ceil(0.01*dict[-1])
    fivePercent = math.ceil(0.05*dict[-1])
    tenPercent = math.ceil(0.1*dict[-1])
    twentyPercent = math.ceil(0.2*dict[-1])
    fiftyPercent = math.ceil(0.5*dict[-1])

    holes["~NBR_USERS~"] = dict[-1]
    holes["~YEAR_RANK~"] = dict[0]
    holes["~MONTH_RANK~"] = dict[1]
    holes["~WEEK_RANK~"] = dict[2]
    holes["~DAY_RANK~"] = dict[3]

    if dict[0]<= onePercent: 
        holes["~YEAR_STRING~"] = """<div class="rank onepercent" id="yearRank">
                                        You are in the top 1% this year ! Congratulations !
                                    </div>"""
    elif onePercent<dict[0]<=fivePercent:
        holes["~YEAR_STRING~"] = """<div class="rank fivepercent" id="yearRank">
                                        You are in the top 5% this year ! That's good !
                                    </div>"""
    elif fivePercent<dict[0]<=tenPercent:
        holes["~YEAR_STRING~"] = """<div class="rank tenpercent" id="yearRank">
                                        You are in the top 10% this year ! Keep going !
                                    </div>"""
    elif tenPercent<dict[0]<=twentyPercent:
        holes["~YEAR_STRING~"] = """<div class="rank twentypercent" id="yearRank">
                                        You are in the top 20% this year ! Keep going !
                                    </div>"""
    elif twentyPercent<dict[0]<=fiftyPercent:
        holes["~YEAR_STRING~"] = """<div class="rank fiftypercent" id="yearRank">
                                        You are in the top 50% this year ! That's not bad, but you can do better !
                                    </div>"""
    else:
        holes["~YEAR_STRING~"] = """<div class="rank" id="yearRank">
                                        You did not do very well this year compared to others... maybe try to limit your usage of the server to what's really needed ?
                                    </div>"""

    if dict[1]<= onePercent: 
        holes["~MONTH_STRING~"] = """<div class="rank onepercent" id="monthRank">
                                        You are in the top 1% this month ! Congratulations !
                                    </div>"""
    elif onePercent<dict[1]<=fivePercent:
        holes["~MONTH_STRING~"] = """<div class="rank fivepercent" id="monthRank">
                                        You are in the top 5% this month ! That's good !
                                    </div>"""
    elif fivePercent<dict[1]<=tenPercent:
        holes["~MONTH_STRING~"] = """<div class="rank tenpercent" id="monthRank">
                                        You are in the top 10% this month ! Keep going !
                                    </div>"""
    elif tenPercent<dict[1]<=twentyPercent:
        holes["~MONTH_STRING~"] = """<div class="rank twentypercent" id="monthRank">
                                        You are in the top 20% this month ! Keep going ! 
                                     </div>"""
    elif twentyPercent<dict[1]<=fiftyPercent:
        holes["~MONTH_STRING~"] = """<div class="rank fiftypercent" id="monthRank">
                                        You are in the top 50% this month ! That's not bad, but you can do better !
                                    </div>"""
    else:
        holes["~MONTH_STRING~"] = """<div class="rank" id="monthRank">
                                        You did not do very well this month compared to others... maybe try to limit your usage of the server to what's really needed ?
                                    </div>"""


    if dict[2]<= onePercent: 
        holes["~WEEK_STRING~"] = """<div class="rank onepercent" id="weekRank">
                                    You are in the top 1% this week ! Congratulations !
                                </div>"""
    elif onePercent<dict[2]<=fivePercent:
        holes["~WEEK_STRING~"] = """<div class="rank fivepercent" id="weekRank">
                                    You are in the top 5% this week ! That's good !
                                </div>"""
    elif fivePercent<dict[2]<=tenPercent:
        holes["~WEEK_STRING~"] = """<div class="rank tenpercent" id="weekRank">
                                        You are in the top 10% this week ! Keep going !
                                    </div>"""
    elif tenPercent<dict[2]<=twentyPercent:
        holes["~WEEK_STRING~"] = """<div class="rank twentypercent" id="weekRank">
                                        You are in the top 20% this week ! Keep going !
                                    </div>"""
    elif twentyPercent<dict[2]<=fiftyPercent:
        holes["~WEEK_STRING~"] = """<div class="rank fiftypercent" id="weekRank">
                                    You are in the top 50% this week ! That's not bad, but you can do better !
                                    </div>"""
    else:
        holes["~WEEK_STRING~"] = """<div class="rank" id="weekRank">
                                        You did not do very well this week compared to others... maybe try to limit your usage of the server to what's really needed ?
                                    </div>"""
    
    
    
    if dict[3]<= onePercent: 
        holes["~DAY_STRING~"] = """<div class="rank onepercent" id="dayRank">
                                    You are in the top 1% today ! Congratulations !
                                    </div>"""
    elif onePercent<dict[3]<=fivePercent:
        holes["~DAY_STRING~"] = """<div class="rank fivepercent" id="dayRank">
                                        You are in the top 5% today ! That's good !
                                    </div>"""
    elif fivePercent<dict[3]<=tenPercent:
        holes["~DAY_STRING~"] = """<div class="rank tenpercent" id="dayRank">
                                        You are in the top 10% today ! Keep going !
                                    </div>"""
    elif tenPercent<dict[3]<=twentyPercent:
        holes["~DAY_STRING~"] = """<div class="rank twentypercent" id="dayRank">
                                        You are in the top 20% today ! Keep going !
                                    </div>"""
    elif twentyPercent<dict[3]<=fiftyPercent:
        holes["~DAY_STRING~"] = """<div class="rank fiftypercent" id="dayRank">
                                        You are in the top 50% today ! That's not bad, but you can do better !
                                    </div>"""
    else:
        holes["~DAY_STRING~"] = """<div class="rank" id="dayRank">
                                        You did not do very well today compared to others... maybe try to limit your usage of the server to what's really needed ?
                                    </div>"""

def displayPieChart(sumValues, id):
    values, labels = [],[]
    for elt in sumValues:
        if(elt[1]!=0):
            labels.append(elt[0])
            values.append(elt[1])
    
    plt.pie(values, labels=labels, labeldistance=1.15, wedgeprops={"linewidth":2, "edgecolor":"white"})
    plt.savefig(__file__.replace("client.py", "").replace("\\", "/")+"images/u"+str(id)+"PieChart.png")

    plt.show()
    holes["~PIE_CHART_IMG_SRC~"] = "./images/u"+str(id)+"PieChart.png"



def copyTemplate(infilePath, outfilePath):

    infile = open(infilePath, "r")
    outfile = open(outfilePath, "w")
    for line in infile:
        outfile.write(line)

    infile.close()
    outfile.close()
    




def writeHTML(dict, file):
    html = """
<hr>
<div>
    <h3 style="text-align:center">Consumption on ~SERVER_URL~</h3>
    <p>Today the moment you have consumed the least was at ~TIME_MIN_CONS~ with ~VALUE_MIN_CONS~ mWh, 
    whereas the moment you have consumed the most was at ~TIME_MAX_CONS~ with ~VALUE_MAX_CONS~ mWh.
    Overall, your total consumption in these last 24h was of ~SUM_CONS~ mWh</p>

    <p>This is the consumption of user ~USER_ID~ on ~SERVER_URL~ :</p>
    <div class="grid">
        <div class="row">
            <div class="plot-graph">
                <img src="~GRAPH_IMAGE_SRC~" title="Energy consumption of user number ~USER_ID~. This guy really doesn't care about ecology IMHO...">
            </div>
            <div class="energy-equivalents">
                <div class="energy-info">
                    <p><strong class="energy-value">~SUM_CONS~ mWh</strong> is equivalent to </p>
                        <div><ul>
                            <li>~PHONE_CHARGES_EQ~ phone charges</li>
                            <li>~TV_HOURS_EQ~h of watching TV</li>
                            <li>~KM_EQ~km with an electric car</li>
                            <li>and finally, ~PRICE_EQ~€</li>
                        </ul></div>
                </div>
                <div class="button-div">
                    <button onclick="test()">Click to see the carbon emission of this website !</button>
                </div>
            </div>
        </div>
        <div class="row">
            <div>
                <img src="~BAR_IMAGE_SRC~">
            </div>
            <div> 
                <img src="~WEEKLY_BAR_IMAGE_SRC~" title="Here is your weekly consumption since the start of the monitoring">
            </div>
        </div>
        <p>Below is your ranking over the last day, week, month and year among all the other users of this server. The less energy you consume, the best you are ranked ! </p>
        <div class="row">
            ~DAY_STRING~
            ~WEEK_STRING~
            ~MONTH_STRING~
            ~YEAR_STRING~
        </div>

        
    </div>
</div>
    """
    for src, target in dict.items():
        print(src, target)
        html = html.replace(src, str(target))
    
    file.write(html)
        

    
def addPieChart(dict, file):
    html = """
<hr>
<h2 style="text-align:center">Servers comparison</h3>
<p>With this pie chart you can see how the different servers that you consulted compare with one another.<br>
<b>NB : </b><em>Note that this is based on the daily consumption, and not on the overall consumption.</em>
</p>
<img src="~PIE_CHART_IMG_SRC~" id="piechart">
"""
    for src, target in dict.items():
        print(src, target)
        html = html.replace(src, str(target))
    
    file.write(html)









# Prix du kWh (1kWh = 3 600 000J) pour EDF en février 2025 : 0.2016€ selon https://www.kelwatt.fr/fournisseurs/edf/comprendre-tarif-reglemente-electricite
# Un ordinateur fixe de bureau consomme 123 kWh/an pour 3h45 par jour allumé en moyenne
# Avec 1kWh : TV ~ 4h, four micro-ondes 1h, frigo 1 journée, 1/2 douche ou 1/4 bain, 1 cycle de lavage de linge
#             éclairage pour une journée (avec lampes basses consommation), 2km avec une smart électrique,
#             1/2 journée avec ordi fixe, 1 et 1/2 pour ordi portable
#
# Pile AAA : 1,2 Wh. Combustion d'un litre de carburant diesel produit 10 kWh. Entre 13 et 20 Wh pour charger un tel

if __name__ == "__main__":
    inputMsg = "Please add one or more urls to collect data from or press enter to stop :\n"
    urls = []
    url = input(inputMsg)
    while(url != "" ):
        urls.append(url)
        urls.append(input("Enter your id for this site (integer) :"))
        url = input(inputMsg)
    copyTemplate(__file__.replace("client.py", "websiteTemplate.htm"), __file__.replace("client.py", "website.htm"))
    #copyTemplate("./websiteTemplate.htm", "./website.htm") #When executed from linux
    sumValues = []
    file = open(__file__.replace("client.py", "website.htm"), "a")

    for i in range(0,len(urls),2):
        displayClassicGraph(urls[i], urls[i+1])
        displayTodayBarChart(urls[i], urls[i+1])
        displayWeeklyBarChart(urls[i], urls[i+1])
        displayRankings(urls[i], urls[i+1])
        print(holes)
        sumValues.append((holes["~SERVER_URL~"], holes["~SUM_CONS~"]))
        writeHTML(holes, file)
    
    displayPieChart(sumValues, urls[1])
    addPieChart(holes, file)
    file.write("""
</body>
</html>
               """)
    file.close()


    