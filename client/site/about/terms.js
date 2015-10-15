var fs = require('fs')
var React = require('react')

var common = require('../common')


module.exports = (
  <common.MainPage title="euphoria: terms of service" nav={<common.PolicyNav selected="terms" />}>
    <common.Markdown className="policy" content={fs.readFileSync(__dirname + '/terms.md', 'utf8')} />
  </common.MainPage>
)
