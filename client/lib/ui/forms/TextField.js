import React from 'react'
import classNames from 'classnames'

import FieldLabelContainer from './FieldLabelContainer'


export default React.createClass({
  displayName: 'TextField',

  propTypes: {
    name: React.PropTypes.string.isRequired,
    label: React.PropTypes.string.isRequired,
    value: React.PropTypes.string,
    onModify: React.PropTypes.func,
    onValidate: React.PropTypes.func,
    onFocus: React.PropTypes.func,
    onBlur: React.PropTypes.func,
    error: React.PropTypes.bool,
    autoFocus: React.PropTypes.bool,
    message: React.PropTypes.string,
    className: React.PropTypes.string,
    inputType: React.PropTypes.string,
    tabIndex: React.PropTypes.number,
    spellCheck: React.PropTypes.bool,
    disabled: React.PropTypes.bool,
  },

  onChange(ev) {
    this.props.onModify(ev.target.value)
  },

  onFocus(ev) {
    if (this.props.onFocus) {
      this.props.onFocus(ev)
    }
  },

  onBlur(ev) {
    this.props.onValidate()
    if (this.props.onBlur) {
      this.props.onBlur(ev)
    }
  },

  focus() {
    this.refs.input.focus()
  },

  render() {
    return (
      <FieldLabelContainer
        className={classNames('text-field', this.props.className)}
        label={this.props.label}
        error={this.props.error}
        message={this.props.message}
      >
        <input
          ref="input"
          name={this.props.name}
          type={this.props.inputType}
          value={this.props.value}
          tabIndex={this.props.tabIndex}
          autoFocus={this.props.autoFocus}
          spellCheck={this.props.spellCheck}
          disabled={this.props.disabled}
          onChange={this.onChange}
          onFocus={this.onFocus}
          onBlur={this.onBlur}
        />
      </FieldLabelContainer>
    )
  },
})
