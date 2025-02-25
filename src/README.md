# Service Energy Monitor

The goal of this project is to build a way for users to monitor their energy consumption of multiple servers. 
In order to do that the server must provide an API to allow the user to access the energy consumption data of the underlying hardware, whereas the user have a python script allowing them to query said APIs and process the data into a static webpage. 


## Doc
The diagrams folder contains a visual explanation of the service, and some slides that sum up the project and the implementation.
#### The Go_code folder
This folder contains all of the code. Let's see what each component is, and how to use them. 
<details>
	<summary> <strong> client folder </strong> </summary>	

#### Files: 
['client.css, client.js'](./Go_code/client/client.css) : two files used when creating the final webpage of the client

['client.py'](./Go_code/client/client.py) : the python scrpit responsible for querying the APIs, processing data into (very) beautiful graphs and creating the webpage that is displayed in the end

['fibonacci.go'](./Go_code/client/fibonacci.go) : this bad implementation of fibonacci sequence is used to simulate activity inside the server and create some changes in energy consumption.

['websiteTemplate.htm'](./Go_code/client/websiteTemplate.htm) : this is the base that client.py will use when creating the final html file. 

['website.htm'](./Go_code/client/website.htm) : this is the final file that will be displayed in the client's browser. It is rewritten each time client.py is run. 

['images/'](./Go_code/client/images) : this is the folder where every image created by the python file will be stored, and it is also where the website.htm file will go looking when it needs to display an image. 

</details>
<details>
	<summary> <strong> config folder </strong> </summary>	

#### Files: 
['config.go'](./Go_code/config/config.go) : this file contains all the constants about databases, that's where you should put your usernames, passwords, urls relative to your own databases. 

</details>
<details>
	<summary> <strong> controller folder </strong> </summary>	

#### Files: 
['controller.go'](./Go_code/controller/controller.go) : this is one of the core elements of this project. The controller file is responsible for querying data from the model, process it, and return it as gin handler functions that are called when accessing the right endpoints. 

['linuxConsumption.go'](./Go_code/controller/linuxConsumption.go) : this file is responsible for getting the energy consumption of the hardware when running on linux machines. It uses the data provided by intel-rapl, so the hardware needs to have this feature. Also because of this, it needs root privileges which means you have to compile the whole project and launch it with <em>sudo ./main</em>. More on this below on the how to use paragraph. 

['readCSV.go'](./Go_code/controller/readCSV.go) : this file was originally thought in order to use this project with ['DEMETER'](https://github.com/Constellation-Group/Demeter) and to base the data consumption on DEMETER csv files. However, it can basically work with any csv given some conditions : it needs to be ";" separated values instead of "," (this is a single character to change in the code, so in reality it's not a big deal), the first column needs to be the UNIX time when the data was retrieved, the last column needs to be the total amount of energy consumed (in mWh) by the concerned process, and finally the last row of each batch of data must end with a row with the name 'CPU Energy' on the second column. 

</details>
<details>
  <summary><strong> model folder</strong></summary>
  
#### Files:
['getPostgres.go'](./Go_code/model/getPostgres.go) : contains all the functions to retrieve data from the postgres database, for example functions to get users, time ranges, or links. 

['updatePostgres.go'](./Go_code/model/updatePostgres.go) : this one also take care of the postgres database, but this time it contains functions to modify the database (reseting the db, creating the tables, deleting users, logging users connection...)

['modelInflux.go'](./Go_code/model/modelInflux.go) : you will find in this file all that is needed to retrieve specific data from the Timeseries Influx database (used to store the energy values, if you needed a reminder).
</details>
<details>
  <summary><strong> view folder</strong></summary>

  #### Files:   
  ['routes.go'](./Go_code/view/routes.go) : its only role is to create the endpoints necessary when simulating a server, and to associate the right functions with the right endpoints.
</details>


## Grid5000 installation
#### Installing dependencies

Postgresql can be installed directly on the node you will reserve. This makes things easier, but it also means you have to do it everytime you want to start an experiment and that you will lose all database between two experiments. 

As for influxdb you can download this ['archive'](./dependencies/influxdb2-2.7.11_linux_amd64.tar.gz) (that you can also find on their website). 
I also recommend you download their CLI package (the last two lines of the shell code below). 
```sh
tar xvzf ./influxdb2-2.7.11_linux_amd64.tar.gz
wget https://dl.influxdata.com/influxdb/releases/influxdb2-client-2.7.5-linux-amd64.tar.gz
tar xvzf ./influxdb2-client-2.7.5-linux-amd64.tar.gz
```
You should now have two directories inside your grid5000 account : influxdb2-2.7.11 and influx

Launch the database, and then go through the setup process. The organization name and token will be used in the code, so remember them !

Next thing to do is create an influxdb token that will serve as authentification. 

Be careful to copy it somewhere safe, or you will have to create another one.

Here we are created one with all-access, a localhost running on port 8086 (default port for influxdb) and an organisation name of "poc" (for proof of concept) in my case, but i strongly advise you pick a personal one. 

```sh
./influxdb2-2.7.11/usr/bin/influxd
./influx setup
./influx auth create --all-access --host http://localhost:8086 --org poc
```

You are now ready to start your influxdb ! Try running the following command : 

```sh
./influxdb2-2.7.11/usr/bin/influxd
```

---
You might also want to install go in your user directory to be able to run go files without compiling them first, or to build the project inside grid5000 instead of building it on your computer and `scp` it into grid5000.

You can either go find the version you want on their website or download [this one](./dependencies/go1.24.0.linux-amd64.tar.gz). After you extract it inside your user directory, you should have a go folder created. 
```sh 
tar -xf go1.24.0.linux-amd64.tar.gz
```
In order to use the go command, you need to run this two commands (everytime you log into gri5000).
```sh
export PATH=$PATH:$HOME/go/bin
export GOPATH=$HOME
```
To verify it works, try typing '$go version' inside the shell. 
If everything works, you should be able to run .go files with 'go run myfile.go', and to build your project : 
```sh
$go build -o main main.go
```


---

#### Now let's get started for real 

Start by reserving a node in interactive mode 

```sh
oarsub -l host=1 -I
```

Then run this commands to install postgresql 

```sh
sudo-g5k apt install postgresql
sudo-g5k su - postgres
createdb journalisation
createuser -s <username>
psql journalisation
```

After you accepted the downloading of postgresql and everything is done, you should logged inside the journalisation database you just created with the user postgres. 

I will use this user to connect to this db, but as you can see with the createuser command, it is possible to choose another user. 

```sh
alter user postgres password 'password';
```
In the config file, the default password for postgres is 'password', but you can obviously change that. 
Now type <b>exit</b> and you should return to your usual username. 
Congrats, you were able to setup the two databases needed for the experiment ! 
Now it is time to run the main file, and to put that server to use !
```sh
sudo-g5k ./main
```