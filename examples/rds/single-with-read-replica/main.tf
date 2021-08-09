# Get the VPC where RDS instance will be created
data "sbercloud_vpc" "vpc_01" {
  name = "put_here_the_name_of_your_existing_vpc"
}

# Get the subnet where RDS instance will be created
data "sbercloud_vpc_subnet" "subnet_01" {
  name = "put_here_the_name_of_your_existing_subnet"
}

# Get the security group for RDS instance 
data "sbercloud_networking_secgroup" "sg_01" {
  name = "put_here_the_name_of_your_existing_security_group"
}

# Get the list of availability zones
data "sbercloud_availability_zones" "list_of_az" {}

# Get RDS flavors
data "sbercloud_rds_flavors" "rds_flavors" {
  db_type       = "MySQL"
  db_version    = "8.0"
  instance_mode = "single"
}

# Get Read Replica flavors
data "sbercloud_rds_flavors" "replica_flavors" {
  db_type       = "MySQL"
  db_version    = "8.0"
  instance_mode = "replica"
}

locals {
  rds_flavor = compact([
    for item in data.sbercloud_rds_flavors.rds_flavors.flavors :
    item["vcpus"] == "2" && item["memory"] == 4 ? item["name"] : ""
  ])[0]
  replica_flavor = compact([
    for item in data.sbercloud_rds_flavors.replica_flavors.flavors :
    item["vcpus"] == "2" && item["memory"] == 8 ? item["name"] : ""
  ])[0]
}

# Create RDS instance
resource "sbercloud_rds_instance" "rds_01" {
  name                  = "terraform-mysql-single"
  flavor                = local.rds_flavor
  vpc_id                = data.sbercloud_vpc.vpc_01.id
  subnet_id             = data.sbercloud_vpc_subnet.subnet_01.id
  security_group_id     = data.sbercloud_networking_secgroup.sg_01.id
  availability_zone     = [data.sbercloud_availability_zones.list_of_az.names[0]]

  db {
    type     = "MySQL"
    version  = "8.0"
    password = "put_here_the_root_user_password"
  }

  volume {
    type = "HIGH"
    size = 80
  }
}

# Create read replica
resource "sbercloud_rds_read_replica_instance" "rds_01_rr_01" {
  name                  = "terraform-mysql-rr"
  flavor                = local.replica_flavor
  primary_instance_id   = sbercloud_rds_instance.rds_01.id
  availability_zone     = data.sbercloud_availability_zones.list_of_az.names[1]

  volume {
    type = "HIGH"
  }
}
