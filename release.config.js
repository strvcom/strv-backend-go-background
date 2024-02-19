'use strict'

module.exports = {
  tagFormat: 'v${version}',
  branches: [
    { name: 'master' },
  ],

  plugins: [
    '@semantic-release/commit-analyzer',
    '@semantic-release/release-notes-generator',
    '@semantic-release/github',
  ],
}
