# freenom-ddns
auto-renew domain name
---
# How to use it

## Edit config.toml
Please type Freenom account(s)
``` toml
[System]
Account = "admin"
Password = "admin"
ReNewTiming = 14400
DdnsTiming = 5

[[Accounts]]
Username = "xxx@xxx.com"
Password = "xxx"

[[Accounts]]
Username = "ooo@ooo.com"
Password = "ooo"
```

## Launch freenom-ddns

On linux
``` sh
./freenom-ddns
```
It will start http service on server, So you may check the status page of FrenomBot on http://localhost:8080
