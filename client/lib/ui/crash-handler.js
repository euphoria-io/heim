var fs = require('fs')
var React = require('react')


var crashedSVG = 'data:image/svg+xml;base64,' + fs.readFileSync(__dirname + '/../../res/crashed.svg', 'base64')

var CrashDialog = React.createClass({
  displayName: 'CrashDialog',

  render: function() {
    // inline and a bit archaic to increase browser compatibility
    var ravenStatus
    if (this.props.ravenEventId) {
      ravenStatus = <p style={{color: 'green'}}>saved an error report. please send us this code: <strong><code>{this.props.ravenEventId}</code></strong></p>
    } else if (this.props.ravenEventId === false) {
      ravenStatus = <p style={{color: '#a20000'}}>failed to send an error report.</p>
    } else {
      ravenStatus = <p style={{color: 'gray'}}>sending an error report &hellip;</p>
    }

    var fontFamily = 'Droid Sans, sans-serif'
    return (
      <div className="mask" style={{
        display: 'table',
        position: 'absolute',
        left: 0,
        right: 0,
        top: 0,
        bottom: 0,
        width: '100%',
        height: '100%',
        background: 'rgba(70, 70, 70, .5)',
      }}>
        <div style={{
          display: 'table-cell',
          verticalAlign: 'middle',
          textAlign: 'center',
        }}>
          <div className="crash-message" style={{
            display: 'inline-block',
            textAlign: 'center',
            fontFamily: fontFamily,
            width: '75%',
            maxWidth: 450,
            padding: '30px 60px',
            lineHeight: '1.35em',
            background: 'rgba(240, 240, 240, .95)',
            borderRadius: 10,
            boxShadow: '0 3px 20px rgba(0, 0, 0, .15)',
          }}>
            <img src={crashedSVG} alt="euphoria crashed" style={{width: 64, height: 64}} />
            <h1 style={{
              fontSize: 22,
              color: '#a20000',
              margin: 10,
            }}>sorry, euphoria had an <span style={{whiteSpace: 'nowrap'}}>error :(</span></h1>
            <p>we'd like to help. if this is happening frequently, please let us know in <a href={process.env.HEIM_PREFIX + '/room/heim'}>&amp;heim</a> or <a href="mailto:hi@euphoria.io">send us an email</a>.</p>
            <div style={{height: '4em'}}>{ravenStatus}</div>
            <button onClick={this.props.onReload} style={{
              fontSize: 24,
              fontFamily: fontFamily,
              background: '#ccc',
              border: 'none',
              padding: '5px 10px',
              borderRadius: 3,
              marginRight: 10,
            }}>reload (recommended)</button>
            <button onClick={this.props.onIgnore} style={{
              fontSize: 24,
              fontFamily: fontFamily,
              background: '#ddd',
              border: 'none',
              padding: '5px 10px',
              borderRadius: 3,
            }}>ignore</button>
          </div>
        </div>
      </div>
    )
  },
})

module.exports = function(ev) {
  if (uidocument.getElementById('crash-dialog')) {
    return
  }

  var container = uidocument.createElement('div')
  container.id = 'crash-dialog'

  var component = <CrashDialog
    onReload={() => uiwindow.location.reload()}
    onIgnore={() => container.parentNode.removeChild(container)}
  />
  var crashDialog = React.render(component, container)
  uidocument.body.appendChild(container)

  function onRavenSent(responseEv) {
    ev.srcElement.removeEventListener('ravenSuccess', onRavenSent, false)
    ev.srcElement.removeEventListener('ravenFailure', onRavenSent, false)
    var ravenEventId = false
    if (responseEv.type == 'ravenSuccess') {
      // jshint camelcase: false
      ravenEventId = responseEv.data.event_id
    }
    crashDialog.setProps({ravenEventId: ravenEventId})
  }

  ev.srcElement.addEventListener('ravenSuccess', onRavenSent, false)
  ev.srcElement.addEventListener('ravenFailure', onRavenSent, false)
}
