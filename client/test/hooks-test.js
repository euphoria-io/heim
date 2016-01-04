require('./support/setup')
import assert from 'assert'

import Hooks from '../lib/Hooks'


describe('Hooks', () => {
  it('should create hooks named in the constructor arguments', () => {
    const hooks = new Hooks('a', 'b', 'c')
    assert.deepEqual(hooks._hooks, {
      'a': [],
      'b': [],
      'c': [],
    })
  })

  describe('create', () => {
    it('should add a hook to the index', () => {
      const hooks = new Hooks()
      assert.deepEqual(hooks._hooks, {})
      hooks.create('z')
      assert.deepEqual(hooks._hooks, {
        'z': [],
      })
    })
  })

  describe('register', () => {
    it('should add a callback to a hook', () => {
      function callback() {}
      const hooks = new Hooks('x')
      hooks.register('x', callback)
      assert.deepEqual(hooks._hooks, {
        'x': [callback],
      })
    })
  })

  describe('run', () => {
    it('should run all callbacks registered to a hook in order and return the results', () => {
      const hooks = new Hooks('x')
      hooks.register('x', () => 1)
      hooks.register('x', () => 2)
      hooks.register('x', () => 3)
      assert.deepEqual(hooks.run('x'), [1, 2, 3])
    })
  })
})
