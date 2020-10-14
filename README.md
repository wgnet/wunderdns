# WunderDNS

WunderDNS is a **multitenant api for PowerDNS PostgreSQL database**. Again, it's a API for PostgreSQL database, not to PowerDNS itself. It is used to alter DNS records in Wargaming since 2016 ( it was called sync_powerdns2.pl though ) and now it was reborned as a golang application.
## Key features
- **Multitenancy** - if you create a record, nobody else can change it
- **ACLs** - you may configure any combinations of domains and permissions
- **HTTP API** - simple way to get access
- **AMQP API** - a way to get your requests delivered
- **Flawless integration** - you even don't need to alter your powerdns server or postgresql database to start using wunderdns
- **Multiple databases support** - you may alter a `few` databases in one request
- **Two views support** - wunderdns supports both public & private `views` to separate local & public records

## How it works
WunderDNS connects to PostgreSQL database and works directly with PowerDNS tables: records, domains & so on. It uses their own table records_api ( inherits records ) to alter records. WunderDNS uses AMQP to get requests and send replies. It also uses HTTP API that works as a AMQP<->HTTP gateway.

## Dependencies
- [PowerDNS](https://www.powerdns.com/) with PostgreSQL database
- [RabbitMQ](https://www.rabbitmq.com/) cluster
- [golang](https://golang.org/) >= 1.11

## Installation
- make
- powerdns\# CREATE TABLE records_api(owner VARCHAR(255)) INHERITS(records);
- configure
- run

## Configuration
See [wunderdns.ini](wunderdns.ini) and [wunderapi.ini](wunderapi.ini) for details.

## Usage

//TODO

## Known issues

[ISSUES.md](ISSUES.md)

## Getting help

Create an issue :)

## Getting involved

[CONTRIBUTING](CONTRIBUTING.md).

