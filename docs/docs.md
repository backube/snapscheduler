# Editing the documentation

The documentation is built using [Jekyll](https://jekyllrb.com/) and hosted via
[GitHub Pages](https://pages.github.com/).

All content is written as Markdown within the `/docs` directory, and these files
get transformed into html and hosted at
`https://backube.github.io/snapscheduler/`.

## Locally viewing the content

When editing the documentation, the rendered content can be viewed locally by
using Jekyll.

First, install Ruby and Bundler:

 ```
sudo dnf install ruby ruby-devel @development-tools
gem install bundler
```

Update the Gems:

```
bundler update
```

Build and serve the documentation:

```
PAGES_REPO_NWO=backube/snapscheduler bundle exec jekyll serve -l
```

The above command will display a URL that contains the root of the site.

As you make modfications to the `*.md` files, the content should be updated
automatically--- though there seem to be some exceptions. In the case of new
files or changes the the `_config.yml`, it may be necessary to restart jekyll.

## Publishing changes

When new changes are committed to the repo, they are automatically picked up and
deployed by the standard GitHub Pages mechanisms. While the source files get
linted via Travis CI, all rendering and deployment is handled via GitHub.
