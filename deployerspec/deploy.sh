#!/bin/bash -eu

export TF_IN_AUTOMATION=1

statefile="/state/terraform.tfstate"
planFilename="/work/update.plan"

# the zip name needs to change from previous deployment for it to be considered new
function renameZipToUniqueFilename {
	local newZipName="lambdafunc-$FRIENDLY_REV_ID.zip"

	if [ ! -e "$newZipName" ]; then
		ln -s "lambdafunc.zip" "$newZipName"
	fi

	echo "zip_filename = \"$newZipName\"" > terraform.tfvars
}

function setupTerraform {
	# needed for Terraform to resolve modules
	terraform get
}

function generateUpdatePlan {
	terraform plan -state "$statefile" -out "$planFilename"
}

function terraformApply {
	# we can't give "-state" flag to apply verb if we're using an execution plan, and Terraform
	# will save to current workdir which is fucking different than what was given in plan verb...
	terraform apply -state-out "$statefile" "$planFilename"
}

renameZipToUniqueFilename

setupTerraform

generateUpdatePlan

# wait for enter
echo "[press any key to deploy or ctrl-c to abort]"
read DUMMY

terraformApply
