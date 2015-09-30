var React = require('react')

var common = require('./common')


module.exports = (
  <common.Page title="euphoria!" className="welcome">
    <div className="clicky">
      <a className="logo" href={common.heimURL('/room/welcome')}>welcome</a>
      <div className="colors">
        <div className="a"></div>
        <div className="b"></div>
        <div className="c"></div>
        <div className="d"></div>
        <div className="e"></div>
      </div>
    </div>
    <common.Footer />
  </common.Page>
)
