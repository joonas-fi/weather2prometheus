
variable "region" { type = "string" }
variable "WEATHER_ZIPCODE" { type = "string" }
variable "WEATHER_COUNTRYCODE" { type = "string" }
variable "OPENWEATHERMAP_APIKEY" { type = "string" }
variable "PROMPIPE_ENDPOINT" { type = "string" }
variable "PROMPIPE_AUTHTOKEN" { type = "string" }

variable "zip_filename" { type = "string" }

provider "aws" {
	region = "${var.region}"
}

resource "aws_lambda_function" "fn" {
	function_name = "Weather-to-prompipe"
	description = "Delivers weather reports to prompipe endpoint"

	filename = "${var.zip_filename}"

	handler = "weather"
	runtime = "go1.x"

	# FIXME
	role = "arn:aws:iam::329074924855:role/AlertManager"

	timeout = 30

	environment {
		variables = {
			WEATHER_ZIPCODE = "${var.WEATHER_ZIPCODE}"
			WEATHER_COUNTRYCODE = "${var.WEATHER_COUNTRYCODE}"
			OPENWEATHERMAP_APIKEY = "${var.OPENWEATHERMAP_APIKEY}"
			PROMPIPE_ENDPOINT = "${var.PROMPIPE_ENDPOINT}"
			PROMPIPE_AUTHTOKEN = "${var.PROMPIPE_AUTHTOKEN}"
		}
	}
}

resource "aws_cloudwatch_event_rule" "cw_scheduledevent_rule" {
	name = "Weather-schedule"
	description = "Scheduled invokation for Lambda fn"
	schedule_expression = "rate(5 minute)"
}

resource "aws_cloudwatch_event_target" "cwlambdatarget" {
	target_id = "LambdaFnInvoke"
	rule = "${aws_cloudwatch_event_rule.cw_scheduledevent_rule.name}"
	arn = "${aws_lambda_function.fn.arn}"
}

resource "aws_lambda_permission" "cloudwatch_scheduler" {
	statement_id = "AllowExecutionFromCloudWatch"
	action = "lambda:InvokeFunction"
	function_name = "${aws_lambda_function.fn.function_name}"
	principal = "events.amazonaws.com"
	source_arn = "${aws_cloudwatch_event_rule.cw_scheduledevent_rule.arn}"
}
