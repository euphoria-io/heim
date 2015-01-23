require('./support/setup')
var assert = require('assert')
var React = require('react/addons')
var TestUtils = React.addons.TestUtils


describe('<Message>', function() {
  var Tree = require('../lib/tree')
  var Message = require('../lib/ui/message')
  var testTree

  it('automatically links urls', function() {
    testTree = new Tree('time').reset([
      {
        'id': 'id1',
        'time': 123456,
        'sender': {
          'id': '32.64.96.128:12345',
          'name': 'tester',
        },
        'content': 'http://google.com',
      }
    ])

    var message = TestUtils.renderIntoDocument(
      <Message tree={testTree} nodeId="id1" depth={0} />
    )

    var messageContent = TestUtils.findRenderedDOMComponentWithClass(message, 'message')
    assert.equal(messageContent.getDOMNode().innerHTML,
      '<a href="http://google.com" target="_blank" rel="noreferrer">google.com</a>')
  })

  it('truncates long urls', function() {
    var testTree = new Tree('time').reset([
      {
        'id': 'id1',
        'time': 123456,
        'sender': {
          'id': '32.64.96.128:12345',
          'name': 'tester',
        },
        'content': 'http://google.com/abcdefghijklmnopqrstuvwxyz1234567890',
      }
    ])

    var message = TestUtils.renderIntoDocument(
      <Message tree={testTree} nodeId="id1" depth={0} />
    )

    var messageContent = TestUtils.findRenderedDOMComponentWithClass(message, 'message')
    assert.equal(messageContent.getDOMNode().innerHTML,
      '<a href="http://google.com/abcdefghijklmnopqrstuvwxyz1234567890" target="_blank" rel="noreferrer">google.com/abcdefghijklmnopqrstuvwxyz1..</a>')
  })

  it('doesn\'t linkify javascript:// links', function() {
    // note: jshint warns about javascript:// URLs
    var testTree = new Tree('time').reset([
      {
        'id': 'id1',
        'time': 123456,
        'sender': {
          'id': '32.64.96.128:12345',
          'name': 'tester',
        },
        'content': 'Javascript://hello javascript://world',  // jshint ignore:line
      }
    ])

    var message = TestUtils.renderIntoDocument(
      <Message tree={testTree} nodeId="id1" depth={0} />
    )

    var messageContent = TestUtils.findRenderedDOMComponentWithClass(message, 'message')
    assert.equal(messageContent.getDOMNode().innerHTML,
      'Javascript://hello javascript://world')  // jshint ignore:line
  })
})
