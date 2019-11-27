variable "vpc_id" {
  type        = "string"
  description = "ID of the VPC where the vpcinfo infrastructure is deployed."
}

variable "domain" {
  type        = "string"
  default     = "vpc.local"
  description = "Name of the domain name under which the VPC information will be made available."
}

variable "max_subnet_count" {
  type        = "string"
  default     = "32"
  description = "This input variable is used as a workaround for terraform requiring count properties to have a known value at 'compile time', just set it to something larger than the number of subnets in your VPC."
}

data "aws_subnet_ids" "subnets" {
  vpc_id = "${var.vpc_id}"
}

data "aws_subnet" "list" {
  count = "${length(data.aws_subnet_ids.subnets.ids)}"
  id    = "${element(data.aws_subnet_ids.subnets.ids, count.index)}"
}

locals {
  subnet_ids   = "${slice(data.aws_subnet.list.*.id, 0, length(data.aws_subnet_ids.subnets.ids))}"
  subnet_cidrs = "${slice(data.aws_subnet.list.*.cidr_block, 0, length(data.aws_subnet_ids.subnets.ids))}"
  subnet_zones = "${slice(data.aws_subnet.list.*..availability_zone, 0, length(data.aws_subnet_ids.subnets.ids))}"
}

resource "aws_route53_zone" "vpc" {
  name = "${var.domain}"

  vpc {
    vpc_id = "${var.vpc_id}"
  }
}

resource "aws_route53_record" "subnets" {
  zone_id = "${aws_route53_zone.vpc.zone_id}"
  name    = "subnets.${var.domain}"
  ttl     = "60"
  type    = "TXT"

  records = [
    "${formatlist("subnet=%s,cidr=%s,zone=%s", local.subnet_ids, local.subnet_cidrs, local.subnet_zones)}",
  ]
}
