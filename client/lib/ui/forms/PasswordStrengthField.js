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
    minEntropy: React.PropTypes.number,
    showFeedback: React.PropTypes.bool,
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
    let strength = 'unknown'
    if (this.props.minEntropy) {
      const entropy = entropizer.evaluate(value)
      let message
      if (entropy < this.props.minEntropy) {
        strength = 'weak'
        message = 'too simple — add more!'
      } else {
        strength = 'ok'
      }
      this.setState({strength, message})
    }
    this.props.onModify({
      text: value,
      strength: strength,
    })
  },

  focus() {
    this.refs.field.focus()
  },

  render() {
    const strengthClass = this.props.showFeedback ? this.state.strength : null
    const strengthMessage = this.props.showFeedback ? this.state.message : null
    let message
    if (!this.props.message || this.state.focused && strengthMessage) {
      message = strengthMessage
    } else {
      message = this.props.message
    }
    return (
      <TextField
        ref="field"
        inputType="password"
        {...this.props}
        value={this.props.value && this.props.value.text}
        className={classNames('password-field', strengthClass)}
        message={message}
        onModify={this.onModify}
        onFocus={this.onFocus}
        onBlur={this.onBlur}
      />
    )
  },
})
