import support from './support/setup'
import assert from 'assert'

import update from '../lib/stores/update'


describe('update store', () => {
  beforeEach(() => {
    support.resetStore(update.store)
  })

  it('should initialize with default state', () => {
    const initialState = update.store.getInitialState()
    assert.equal(initialState.get('ready'), false)
    assert.equal(initialState.get('currentVersion'), null)
    assert.equal(initialState.get('newVersion'), null)
  })

  describe('on chat state change', () => {
    it('should store the current version if none set')
    describe('if the server version changes', () => {
      it('should store the new version, if not seen before')
      it('should prepare to update if the window is focused')
      it('should not prepare to update if the window is not focused')
    })
  })

  describe('on focus state change', () => {
    it('should prepare to update if the window is focused')
  })

  describe('prepare action', () => {
    it('should prepare an update')
    it('should skip preparing the same update twice')
  })

  describe('setReady action', () => {
    it('should update ready state')
    it('should store update finalize callback')
  })

  describe('perform action', () => {
    it('should call stored update finalize callback')
  })
})
