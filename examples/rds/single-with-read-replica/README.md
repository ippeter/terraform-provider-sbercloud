## Example: Single MySQL Instance with Read Replica

### Requirements

- VPC exists in SberCloud.Advanced
- subnet exists in SberCloud.Advanced
- security group exists in SberCloud.Advanced

### Description

This example provisions a single MySQL instance and one read replica with the following attributes.

**Database**
- DB engine: MySQL
- version: 8.0
- DB type: single node
- availability zone: ru-moscow-1a
- instance flavor: rds.mysql.c6.large.2 (2 vCPUs, 4 GB)
- storage: 80 GB, SAS

**Read replica**
- availability zone: ru-moscow-1b
- instance flavor: rds.mysql.c6.large.4.rr (2 vCPUs, 8 GB)
- storage: 80 GB, SAS

### Notes

Please note the **sbercloud_rds_flavors** data source.  
It gets the list of corresponding flavor names based on the database engine, version and instance mode ("single", "ha", "replica").
It helps avoid hardcoding flavor names in resources.

Then, using local variables, one can filter out flavors with required amount of vCPUs and RAM.

Please also note, that in this example the read replica uses **another** flavor than the database itself, and it's placed in **another** availability zone.
