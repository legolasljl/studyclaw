git pull
go mod tidy
go build ./
nohup ./studyclaw > studyclaw.log 2>&1 & echo $!>pid.pid
