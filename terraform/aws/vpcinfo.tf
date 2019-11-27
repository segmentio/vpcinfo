variable "vpc_id" {
  type        = "string"
  description = "ID of the VPC where the vpcinfo infrastructure is deployed."
}

variable "domain" {
  type        = "string"
  default     = "vpc.local"
  description = "Name of the domain name under which the VPC information will be made available."
}

data "aws_subnet_ids" "subnets" {
  vpc_id = "${var.vpc_id}"
}

data "aws_subnet" "list" {
  count = "${length(data.aws_subnet_ids.subnets.ids)}"
  id    = "${element(data.aws_subnet_ids.subnets.ids, count.index)}"
}

resource "aws_route53_zone" "vpc" {
  name          = "${var.domain}"
  comment       = "DNS zone managed by https://github.com/segmentio/vpcinfo, contains TXT records carrying information about the VPC."
  force_destroy = true

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
    "${formatlist("subnet=%s,cidr=%s,zone=%s", data.aws_subnet.list.*.id, data.aws_subnet.list.*.cidr_block, data.aws_subnet.list.*.availability_zone)}",
  ]
}
