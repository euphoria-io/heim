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
      {id: '1', value: 'hello', time: 5},
      {id: '2', parent: '1', value: 'world', time: 5},
    ]
    var tree

    beforeEach(function() {
      tree = new Tree('time').reset(entries)
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
        tree.add({id: '3', parent: '1', value: 'yo', time: 7})
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

    describe('after adding multiple nodes', function() {
      var entries = [
        {id: '0', value: 'first!', time: 0},
        {id: '3', parent: '1', value: 'local first!', time: 1},
        {id: '9', value: 'last', time: 9},
      ]

      beforeEach(function() {
        tree.addAll(entries)
      })

      function check() {
        it('the node with highest time should be last', function() {
          assert.equal(tree.last(), tree.get('9'))
        })

        it('should only trigger a change event for the parents of new nodes', function() {
          sinon.assert.calledTwice(tree.changes.emit)
          sinon.assert.calledWithExactly(tree.changes.emit, '__root', tree.get('__root'))
          sinon.assert.calledWithExactly(tree.changes.emit, '1', tree.get('1'))
        })

        it('should visit all nodes in a map traversal (in the right order)', function() {
          var visited = tree.mapDFS(debugMap)
          assert.deepEqual(visited, [
            '__root', 0, [
              ['0', 1, []],
              ['1', 1, [
                ['3', 2, []],
                ['2', 2, []],
              ]],
              ['9', 1, []],
            ]
          ])
        })
      }

      check()

      describe('after re-adding the same nodes', function() {
        beforeEach(function() {
          tree.changes.emit.reset()
          tree.addAll(entries)
        })

        describe('should not change', function() {
          check()
        })
      })
    })

    describe('after adding a node with a missing parent', function() {
      beforeEach(function() {
        tree.add({id: '3', parent: 'wtf', value: 'yo', time: 7})
      })

      it('should contain the new node', function() {
        assert(tree.get('3'))
      })

      it('should contain an unreachable parent for the new node', function() {
        var parent = tree.get('wtf')
        assert(parent.get('children').contains('3'))
        assert(!parent.has('parent'))
      })

      it('should update the size', function() {
        assert.equal(tree.size, 3)
      })

      describe('after adding the missing parent', function() {
        beforeEach(function() {
          tree.add({id: 'wtf', parent: '1', value: 'j0', time: 6})
        })

        it('should update the size', function() {
          assert.equal(tree.size, 4)
        })

        it('should visit all nodes in a map traversal', function() {
          var visited = tree.mapDFS(debugMap)
          assert.deepEqual(visited, [
            '__root', 0, [
              ['1', 1, [
                ['2', 2, []],
                ['wtf', 2, [
                  ['3', 3, []]
                ]],
              ]]
            ]
          ])
        })
      })
    })

    describe('after merging an update to a node', function() {
      beforeEach(function() {
        tree.mergeNode('2', {value: 'dawg'})
      })

      it('should update the node', function() {
        assert.equal(tree.get('2').get('value'), 'dawg')
      })

      it('should trigger a change event', function() {
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
