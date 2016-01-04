import _ from 'lodash'


class Hooks {
  constructor(...names) {
    this._hooks = {}
    _.each(names, n => this.create(n))
  }

  create(name) {
    this._hooks[name] = []
  }

  register(name, callback) {
    this._hooks[name].push(callback)
  }

  run(name, context, ...args) {
    return _.map(this._hooks[name], h => h.apply(context, args))
  }
}

export default Hooks  // work around https://github.com/babel/babel/issues/2694
