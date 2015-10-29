export default function displayCrashDialog(ev) {
  if (uidocument.getElementById('crash-dialog')) {
    return
  }

  // defer loading until we are actually rendering a crash dialog (speeds up initial client.js connection)
  const fs = require('fs')
  const React = require('react')
  const crashedSVG = 'data:image/svg+xml;base64,' + fs.readFileSync(__dirname + '/../../res/crashed.svg', 'base64')
  const crashedCSS = fs.readFileSync(__dirname + '/../../build/heim/crashed.css')

  const CrashDialog = React.createClass({
    displayName: 'CrashDialog',

    propTypes: {
      ravenEventId: React.PropTypes.oneOfType([React.PropTypes.string, React.PropTypes.bool]),
      onReload: React.PropTypes.func,
      onIgnore: React.PropTypes.func,
    },

    render() {
      let ravenStatus
      if (this.props.ravenEventId) {
        ravenStatus = <p className="saved">saved an error report. <span style={{whiteSpace: 'nowrap'}}>please send us this code:</span> <strong><code>{this.props.ravenEventId}</code></strong></p>
      } else if (this.props.ravenEventId === false) {
        ravenStatus = <p className="failed">failed to send an error report.</p>
      } else {
        ravenStatus = <p>sending an error report &hellip;</p>
      }

      return (
        <div className="mask">
          <div className="container">
            <div className="crash-message">
              <img className="logo" src={crashedSVG} alt="euphoria crashed" />
              <h1>sorry, euphoria had an <span style={{whiteSpace: 'nowrap'}}>error :(</span></h1>
              <p>we'd like to help. if this is happening frequently, please let us know in <a href={process.env.HEIM_PREFIX + '/room/heim'}>&amp;heim</a> or <a href="mailto:hi@euphoria.io">send us an email</a>.</p>
              <div className="raven-status-container">{ravenStatus}</div>
              <button onClick={this.props.onReload} className="reload">reload (recommended)</button>
              <button onClick={this.props.onIgnore}>ignore</button>
            </div>
          </div>
          <style dangerouslySetInnerHTML={{__html: crashedCSS}} />
        </div>
      )
    },
  })

  const container = uidocument.createElement('div')
  container.id = 'crash-dialog'

  const component = (
    <CrashDialog
      onReload={() => uiwindow.location.reload()}
      onIgnore={() => container.parentNode.removeChild(container)}
    />
  )
  React.render(component, container)
  uidocument.body.appendChild(container)

  function onRavenSent(responseEv) {
    ev.srcElement.removeEventListener('ravenSuccess', onRavenSent, false)
    ev.srcElement.removeEventListener('ravenFailure', onRavenSent, false)
    let ravenEventId = false
    if (responseEv.type === 'ravenSuccess') {
      ravenEventId = responseEv.data.event_id
    }
    crashDialog.setProps({ravenEventId: ravenEventId})
  }

  ev.srcElement.addEventListener('ravenSuccess', onRavenSent, false)
  ev.srcElement.addEventListener('ravenFailure', onRavenSent, false)
}
