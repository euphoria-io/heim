import React from 'react'
import classNames from 'classnames'
import Entropizer from 'entropizer'

import TextField from './TextField'


const entropizer = new Entropizer()

export default React.createClass({
  displayName: 'PasswordField',

  propTypes: {
    name: React.PropTypes.string.isRequired,
    label: React.PropTypes.string.isRequired,
    minEntropy: React.PropTypes.number.isRequired,
    value: React.PropTypes.object,
    onModify: React.PropTypes.func,
    onValidate: React.PropTypes.func,
    error: React.PropTypes.bool,
    message: React.PropTypes.string,
    className: React.PropTypes.string,
    tabIndex: React.PropTypes.number,
    disabled: React.PropTypes.bool,
  },

  getInitialState() {
    return {
      focused: false,
      strength: null,
      message: null,
    }
  },

  onFocus() {
    this.setState({focused: true})
  },

  onBlur() {
    this.setState({focused: false})
  },

  onModify(value) {
    const entropy = entropizer.evaluate(value)
    let strength
    let message
    if (entropy < this.props.minEntropy) {
      strength = 'weak'
      message = 'too simple — add more!'
    } else {
      strength = 'ok'
    }
    this.setState({strength, message})
    this.props.onModify({
      value: value,
      strength: strength,
    })
  },

  focus() {
    this.refs.field.focus()
  },

  render() {
    return (
      <TextField
        ref="field"
        inputType="password"
        {...this.props}
        value={this.props.value && this.props.value.value}
        className={classNames('password-field', this.state.strength)}
        message={(this.props.message && !this.state.focused) ? this.props.message : this.state.message}
        onModify={this.onModify}
        onFocus={this.onFocus}
        onBlur={this.onBlur}
      />
    )
  },
})
