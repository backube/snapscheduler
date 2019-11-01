# Documentation for snapscheduler

This directory holds the source for snapscheduler's documentation.

The documentation can be viewed at the github-pages site:
[https://backube.github.io/snapscheduler](https://backube.github.io/snapscheduler)

------

The documentation can be viewed/edited locally using [jekyll](https://jekyllrb.com/).

## Prerequisites

* Install Ruby
  * Fedora: `sudo dnf install ruby ruby-devel @development-tools`
* Install bundler
  * `gem install bundler`

## Serve the docs locally

* Switch to the `/docs` directory
* Install/update the local gems
  * `bundle update`
* Serve the docs
  * `bundle exec jekyll serve -l`
