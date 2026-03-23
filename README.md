## Notes for Tim

Command to open a new tab in WSL with windows terminal

``` bash
wt.exe -w 0 nt --profile "Ubuntu-24.04" -- wsl.exe -- bash -c "top -d 0.5"
```

Create user based on your current user

``` sql
CREATE USER 'stewart'@'localhost' IDENTIFIED WITH auth_socket;
GRANT ALL PRIVILEGES ON *.* TO 'stewart'@'localhost' WITH GRANT OPTION;
FLUSH PRIVILEGES;
```

Comamnd to start mysql.

``` bash
../install/8.4.8/bin/mysqld --datadir=. --user=$(whoami) --skip-grant-tables
```

The follwing will run a command in with mysqlclient built against a non
standard mysql location.

``` bash
LD_LIBRARY_PATH=/home/stewart/code/open-source/mysql/install/8.4.8/lib ./manage.py runserver
```

``` yaml
tabs:
    - top
    - htop
```
