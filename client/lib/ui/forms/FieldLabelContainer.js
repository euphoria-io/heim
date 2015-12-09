import React from 'react'
import classNames from 'classnames'


export default function FieldLabelContainer(props) {
  return (
    <label className={classNames('field-label-container', props.error && 'error', props.className)}>
      <div className="label">{props.label}</div>
      {props.message && <div className="message">{props.message}</div>}
      {props.children}
    </label>
  )
}

FieldLabelContainer.propTypes = {
  label: React.PropTypes.string.isRequired,
  className: React.PropTypes.string,
  error: React.PropTypes.bool,
  message: React.PropTypes.string,
  children: React.PropTypes.node,
}
