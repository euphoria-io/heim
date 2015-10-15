var fs = require('fs')
var React = require('react')

var common = require('../common')


module.exports = (
  <common.MainPage title="euphoria: privacy policy" nav={<common.PolicyNav selected="privacy" />}>
    <common.Markdown className="policy" content={fs.readFileSync(__dirname + '/privacy.md', 'utf8')} />
  </common.MainPage>
)
