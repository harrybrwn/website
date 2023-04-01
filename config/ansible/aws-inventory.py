#!/usr/bin/env python3

import os
import boto3
import json


def get_tag(instance, key) -> str:
	for tag in instance.tags:
		if tag["Key"] == key:
			return tag["Value"]
	return ""


session = boto3.Session(profile_name=os.getenv("AWS_PROFILE", "default"))
ec2 = session.resource("ec2")
instance_filters = [
	{
		"Name": "instance-state-name",
		"Values": ["running"],
	},
	{
		"Name": "tag:Environment",
		"Values": ["dev"],
	}
]

ansible_hosts = []
hostvars = {}
groups = {}

for instance in ec2.instances.filter(Filters=instance_filters):
	name = get_tag(instance, "Name")
	environment = get_tag(instance, "Environment")
	instance_vars = {
		"ansible_host": instance.public_ip_address,
		"environment": environment,
		"aws_id": instance.id,
		"aws_name": name,
	}
	if name in hostvars:
		continue
	hostvars[name] = instance_vars

output = {
	"_meta": {
		"hostvars": hostvars,
	},
	"all": {
		"children": ["ungrouped"] + list(groups.keys())
	}
}
awsdev = []
for name, var in hostvars.items():
	if var["environment"] == "dev":
		awsdev.append(name)

if awsdev:
	gropus["awsdev"] = {"children": awsdev}
output.update(groups)

print(json.dumps(output, indent=4))