## Example: Basic RDS Instance (PostgreSQL)

### Requirements

- VPC exists in SberCloud.Advanced
- subnet exists in SberCloud.Advanced
- security group exists in SberCloud.Advanced

### Description

This example provisions a basic RDS instance with the following attributes:
- DB engine: PostgreSQL
- version: latest
- DB type: single node
- availability zone: ru-moscow-1a
- instance flavor: rds.pg.c6.large.2 (2 vCPUs, 4 GB)
- storage: 60 GB, SSD

