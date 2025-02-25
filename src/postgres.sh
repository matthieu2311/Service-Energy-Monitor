
# These commands can run automatically
export PATH=$PATH:$HOME/go/bin # Allows you to use the go command
export GOPATH=$HOME
sudo-g5k apt install postgresql # Install postgres
sudo-g5k su - postgres # Switch to user postgres to modify postgres
createdb journalisation # Create the postgres database
createuser -s <username> # Gives you the right to modify postgres (-s means superuser)
psql journalisation # Enter the database

# Once inside the journalisation postsgres database, change the password to 'password' (or the one in the config file)
#alter user postgres password 'password';

#To check server status
#sudo-g5k systemctl status postgresql@13-main

#To start the server, must be user postgres
#pg_ctlcluster 13 main start

#To build your go project, run this command inside the repository containing your main.go file.
#go build -o main main.go
