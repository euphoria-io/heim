require('./support/setup')
var assert = require('assert')
var sinon = require('sinon')
var Immutable = require('immutable')

var Tree = require('../lib/tree')

describe('Tree', function() {
  function debugMap(node, children, depth) {
    return [node.get('id'), depth, children.toArray()]
  }

  function checkMapOnlyRoot(tree) {
    var spy = sinon.spy()
    tree.mapDFS(spy)
    sinon.assert.calledOnce(spy)
    sinon.assert.calledWithExactly(spy, tree.index.__root, Immutable.OrderedSet(), 0)
  }

  describe('a new empty tree', function() {
    var tree = new Tree()

    it('should have size 0', function() {
      assert.equal(tree.size, 0)
    })

    it('should only visit root in traversal', function() {
      checkMapOnlyRoot(tree)
    })
  })

  describe('initialized with an array of entries', function() {
    var entries = [
      {id: '1', value: 'hello'},
      {id: '2', parent: '1', value: 'world'},
    ]
    var tree

    beforeEach(function() {
      tree = new Tree(entries)
      sinon.stub(tree.changes, 'emit')
    })

    afterEach(function() {
      tree.changes.emit.restore()
    })

    it('should have correct size', function() {
      assert.equal(tree.size, 2)
    })

    it('should nest nodes with parents', function() {
      assert(tree.get('1').get('children').contains('2'))
    })

    it('should visit all nodes in a map traversal', function() {
      var visited = tree.mapDFS(debugMap)
      assert.deepEqual(visited, [
        '__root', 0, [
          ['1', 1, [
            ['2', 2, []]
          ]]
        ]
      ])
    })

    describe('after adding a node', function() {
      beforeEach(function() {
        tree.add({id: '3', parent: '1', value: 'yo'})
      })

      it('should contain the new node', function() {
        assert(tree.get('1').get('children').contains('3'))
      })

      it('should update the size', function() {
        assert.equal(tree.size, 3)
      })

      it('should trigger a change event on the parent', function() {
        sinon.assert.calledWithExactly(tree.changes.emit, '1', tree.get('1'))
      })

      it('the new node should be last', function() {
        assert.equal(tree.last(), tree.get('3'))
      })

      it('should visit all nodes in a map traversal', function() {
        var visited = tree.mapDFS(debugMap)
        assert.deepEqual(visited, [
          '__root', 0, [
            ['1', 1, [
              ['2', 2, []],
              ['3', 2, []],
            ]]
          ]
        ])
      })
    })

    describe('after adding a node with a missing parent', function() {
      beforeEach(function() {
        tree.add({id: '3', parent: 'wtf', value: 'yo'})
      })

      it('should not contain the new node', function() {
        assert.equal(tree.get('3'), null)
      })

      it('should not update the size', function() {
        assert.equal(tree.size, 2)
      })
    })

    describe('after merging an update to a node', function() {
      beforeEach(function() {
        tree.mergeNode('2', {value: 'dawg'})
      })

      it('should update the node', function() {
        assert.equal(tree.get('2').get('value'), 'dawg')
      })

      it('should only trigger a root change event', function() {
        sinon.assert.calledOnce(tree.changes.emit)
        sinon.assert.calledWithExactly(tree.changes.emit, '2', tree.get('2'))
      })
    })

    describe('after resetting', function() {
      beforeEach(function() {
        tree.reset()
      })

      it('should be empty', function() {
        assert.equal(tree.size, 0)
      })

      it('should only visit root in traversal', function() {
        checkMapOnlyRoot(tree)
      })

      it('should trigger a change event', function() {
        sinon.assert.calledWithExactly(tree.changes.emit, '__root', tree.get('__root'))
      })
    })
  })
})
