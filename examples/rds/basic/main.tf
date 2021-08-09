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

resource "sbercloud_rds_instance" "rds_01" {
  name              = "terraform_pg_single"
  flavor            = "rds.pg.c6.large.2"
  vpc_id            = data.sbercloud_vpc.vpc_01.id
  subnet_id         = data.sbercloud_vpc_subnet.subnet_01.id
  security_group_id = data.sbercloud_networking_secgroup.sg_01.id
  availability_zone = ["ru-moscow-1a"]

  db {
    type     = "PostgreSQL"
    version  = "12"
    password = "put_here_the_root_user_password"
  }

  volume {
    type = "ULTRAHIGH"
    size = 60
  }
}
