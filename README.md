[![Build Status](https://img.shields.io/travis/joonas-fi/weather2prometheus.svg?style=for-the-badge)](https://travis-ci.org/joonas-fi/weather2prometheus)
[![Download](https://img.shields.io/badge/Download-bintray%20latest-blue.svg?style=for-the-badge)](https://bintray.com/joonas/dl/weather2prometheus/_latestVersion#files)

Push weather data to Prometheus from AWS Lambda.

![](docs/grafana.png)

NOTE: currently we're using [prompipe](https://github.com/function61/prompipe) to push
the data, but ideally we'd use Prometheus' pull model.. it's just that the endpoint shouldn't
be polled every 5s to stay within usage quotas.. and our Prometheus autodiscovery doesn't
yet support modifying scrape intervals.


How to deploy
-------------

Follow the same instructions as in [Onni](https://github.com/function61/onni).

```
$ version="..."; deployer deploy weather2prometheus "https://dl.bintray.com/joonas/dl/weather2prometheus/$version/deployerspec.zip"
```
