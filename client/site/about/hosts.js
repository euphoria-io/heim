var fs = require('fs')
var React = require('react')

var common = require('../common')


module.exports = (
  <common.MainPage title="euphoria: room host policy" nav={<common.PolicyNav selected="hosts" />}>
    <common.Markdown className="policy" content={fs.readFileSync(__dirname + '/hosts.md', 'utf8')} />
  </common.MainPage>
)
