require('./support/setup')
var assert = require('assert')


describe('Hooks', function() {
  var Hooks = require('../lib/hooks')

  it('should create hooks named in the constructor arguments', function() {
    var hooks = new Hooks('a', 'b', 'c')
    assert.deepEqual(hooks._hooks, {
      'a': [],
      'b': [],
      'c': [],
    })
  })

  describe('create', function() {
    it('should add a hook to the index', function() {
      var hooks = new Hooks()
      assert.deepEqual(hooks._hooks, {})
      hooks.create('z')
      assert.deepEqual(hooks._hooks, {
        'z': [],
      })
    })
  })

  describe('register', function() {
    it('should add a callback to a hook', function() {
      function callback() {}
      var hooks = new Hooks('x')
      hooks.register('x', callback)
      assert.deepEqual(hooks._hooks, {
        'x': [callback],
      })
    })
  })

  describe('run', function() {
    it('should run all callbacks registered to a hook in order and return the results', function() {
      var hooks = new Hooks('x')
      hooks.register('x', () => 1)
      hooks.register('x', () => 2)
      hooks.register('x', () => 3)
      assert.deepEqual(hooks.run('x'), [1, 2, 3])
    })
  })
})
