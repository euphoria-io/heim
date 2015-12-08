import React from 'react'
import classNames from 'classnames'


export default function FieldLabelContainer(props) {
  return (
    <label className={classNames('field-label-container', props.error && 'error', props.className)}>
      <div className="label-line">
        <div className="label">{props.label}</div>
        {props.action && <button type="button" className="action" onClick={props.onAction}>{props.action}</button>}
        <div className="spacer" />
        {props.message && <div className="message">{props.message}</div>}
      </div>
      {props.children}
    </label>
  )
}

FieldLabelContainer.propTypes = {
  label: React.PropTypes.string.isRequired,
  className: React.PropTypes.string,
  action: React.PropTypes.string,
  onAction: React.PropTypes.func,
  error: React.PropTypes.bool,
  message: React.PropTypes.string,
  children: React.PropTypes.node,
}
