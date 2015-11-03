/* eslint-disable react/prop-types */
// (currently false positive for the nested definition)

export default function displayCrashDialog(ev) {
  if (uidocument.getElementById('crash-dialog')) {
    return
  }

  // defer loading until we are actually rendering a crash dialog (speeds up initial client.js connection)
  const fs = require('fs')
  const React = require('react')
  const ReactDOM = require('react-dom')
  const crashedSVG = 'data:image/svg+xml;base64,' + fs.readFileSync(__dirname + '/../../res/crashed.svg', 'base64')
  const crashedCSS = fs.readFileSync(__dirname + '/../../build/heim/crashed.css')

  function CrashDialog(props) {
    let ravenStatus
    if (props.ravenEventId) {
      ravenStatus = <p className="saved">saved an error report. <span style={{whiteSpace: 'nowrap'}}>please send us this code:</span> <strong><code>{props.ravenEventId}</code></strong></p>
    } else if (props.ravenEventId === false) {
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
            <button onClick={props.onReload} className="reload">reload (recommended)</button>
            <button onClick={props.onIgnore}>ignore</button>
          </div>
        </div>
        <style dangerouslySetInnerHTML={{__html: crashedCSS}} />
      </div>
    )
  }

  CrashDialog.propTypes = {
    ravenEventId: React.PropTypes.oneOfType([React.PropTypes.string, React.PropTypes.bool]),
    onReload: React.PropTypes.func,
    onIgnore: React.PropTypes.func,
  }

  const container = uidocument.createElement('div')
  container.id = 'crash-dialog'

  const component = (
    <CrashDialog
      onReload={() => uiwindow.location.reload()}
      onIgnore={() => container.parentNode.removeChild(container)}
    />
  )
  ReactDOM.render(component, container)
  uidocument.body.appendChild(container)

  function onRavenSent(responseEv) {
    ev.srcElement.removeEventListener('ravenSuccess', onRavenSent, false)
    ev.srcElement.removeEventListener('ravenFailure', onRavenSent, false)
    let ravenEventId = false
    if (responseEv.type === 'ravenSuccess') {
      ravenEventId = responseEv.data.event_id
    }
    const updatedComponent = React.cloneElement(component, {ravenEventId: ravenEventId})
    ReactDOM.render(updatedComponent, container)
  }

  ev.srcElement.addEventListener('ravenSuccess', onRavenSent, false)
  ev.srcElement.addEventListener('ravenFailure', onRavenSent, false)
}
