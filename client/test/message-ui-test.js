require('./support/setup')
var assert = require('assert')
var React = require('react/addons')
var TestUtils = React.addons.TestUtils


describe('<Message>', function() {
  var Tree = require('../lib/tree')
  var Message = require('../lib/ui/message')
  var testTree

  function renderMessage(content) {
    testTree = new Tree('time').reset([
      {
        'id': 'id1',
        'time': 123456,
        'sender': {
          'id': '32.64.96.128:12345',
          'name': 'tester',
        },
        'content': content
      }
    ])

    var message = TestUtils.renderIntoDocument(
      <Message tree={testTree} nodeId="id1" depth={0} />
    )

    return TestUtils.findRenderedDOMComponentWithClass(message, 'message')
  }

  it('automatically links urls', function() {
    var messageContent = renderMessage('http://google.com')
    assert.equal(messageContent.getDOMNode().innerHTML,
      '<a href="http://google.com" target="_blank" rel="noreferrer">google.com</a>')
  })

  it('truncates long urls', function() {
    var messageContent = renderMessage('http://google.com/abcdefghijklmnopqrstuvwxyz1234567890')
    assert.equal(messageContent.getDOMNode().innerHTML,
      '<a href="http://google.com/abcdefghijklmnopqrstuvwxyz1234567890" target="_blank" rel="noreferrer">google.com/abcdefghijklmnopqrstuvwxyz1..</a>')
  })

  it('linkifies &room references', function() {
    var messageContent = renderMessage('hello &space! foo&bar &bar &baz')
    assert.equal(messageContent.getDOMNode().innerHTML,
      'hello <a href="/room/space" target="_blank">&amp;space</a>! foo&amp;bar <a href="/room/bar" target="_blank">&amp;bar</a> <a href="/room/baz" target="_blank">&amp;baz</a>')
  })

  it('doesn\'t linkify javascript:// links', function() {
    // note: jshint warns about javascript:// URLs
    var messageContent = renderMessage('Javascript://hello javascript://world')  // jshint ignore:line
    assert.equal(messageContent.getDOMNode().innerHTML,
      'Javascript://hello javascript://world')  // jshint ignore:line
  })
})
