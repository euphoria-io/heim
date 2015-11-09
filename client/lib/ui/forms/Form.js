const _ = require('lodash')
const React = require('react')
const classNames = require('classnames')


export default React.createClass({
  displayName: 'Form',

  propTypes: {
    context: React.PropTypes.object,
    errors: React.PropTypes.objectOf(React.PropTypes.string),
    validators: React.PropTypes.objectOf(React.PropTypes.func),
    working: React.PropTypes.bool,
    onSubmit: React.PropTypes.func,
    className: React.PropTypes.string,
    children: React.PropTypes.node,
  },

  getDefaultProps() {
    return {
      errors: {},
    }
  },

  getInitialState() {
    return {
      values: {},
      errors: {},
    }
  },

  componentWillMount() {
    this._strict = false
  },

  componentWillReceiveProps(nextProps) {
    if (!_.isEqual(this.props.context, nextProps.context) || !_.isEqual(this.props.validators, nextProps.validators)) {
      this._strict = false
      this.setState({errors: this._validateFields(nextProps.validators, this.state.values, nextProps.context)})
    }
  },

  onFieldModify(name, value) {
    const values = _.assign({}, this.state.values)
    values[name] = value
    this.setState({
      values: values,
      errors: _.assign(this.state.errors, this._validateField(name, values), this._clearError),
    })
  },

  onFieldValidate(name) {
    this.setState({
      errors: _.assign(this.state.errors, this._validateField(name, this.state.values)),
    })
  },

  onSubmit(ev) {
    ev.preventDefault()
    this._strict = true
    const errors = this._validateFields(this.props.validators, this.state.values, this.props.context)
    if (!_.any(errors)) {
      this.setState({errors: {}})
      this._strict = false
      this.props.onSubmit(this.state.values)
    } else {
      this.setState({errors: errors})
    }
  },

  _validateFields(validators, values, context, filter) {
    const errors = {}
    _.each(validators, (validator, fieldSpec) => {
      const fields = fieldSpec.split(' ')
      if (!filter || filter(fields)) {
        _.assign(errors, validator(_.pick(values, fields), this._strict, context))
      }
    })
    return errors
  },

  _validateField(name, values) {
    return this._validateFields(this.props.validators, values, this.props.context, fields => _.contains(fields, name))
  },

  _clearError(origError, newError) {
    return !newError ? null : origError
  },

  _walkChildren(children, errors) {
    return React.Children.map(children, child => {
      if (!React.isValidElement(child)) {
        return child
      } else if (!child.props.name && child.props.type !== 'submit') {
        return React.cloneElement(child, {}, this._walkChildren(child.props.children, errors))
      }

      const name = child.props.name
      const error = name && errors[name]
      return React.cloneElement(child, {
        onModify: value => this.onFieldModify(name, value),
        onValidate: () => this.onFieldValidate(name),
        value: this.state.values[name],
        error: !!error,
        message: error,
        disabled: this.props.working || child.props.type === 'submit' && _.any(errors),
      }, this._walkChildren(child.props.children, errors))
    })
  },

  render() {
    const errors = _.assign({}, this.props.errors, this.state.errors)
    return (
      <form className={classNames('fields', this.props.className)} noValidate onSubmit={this.onSubmit}>
        {this._walkChildren(this.props.children, errors)}
      </form>
    )
  },
})
