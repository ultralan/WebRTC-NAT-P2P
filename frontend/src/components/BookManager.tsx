import { useState, useEffect, useCallback } from 'react'
import { Book } from '../types/book'
import { BookList } from './BookList'
import { BookForm } from './BookForm'

const API_BASE = 'http://localhost:8081'

export function BookManager() {
  const [books, setBooks] = useState<Book[]>([])
  const [editing, setEditing] = useState<Book | null>(null)
  const [showForm, setShowForm] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const fetchBooks = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetch(`${API_BASE}/books`)
      const data = await res.json()
      setBooks(data)
      setError('')
    } catch (e) {
      setError('Failed to fetch books')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchBooks()
  }, [fetchBooks])

  const handleCreate = async (data: Partial<Book>) => {
    try {
      await fetch(`${API_BASE}/books`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      })
      setShowForm(false)
      fetchBooks()
    } catch {
      setError('Failed to create book')
    }
  }

  const handleUpdate = async (data: Partial<Book>) => {
    if (!editing) return
    try {
      await fetch(`${API_BASE}/books/${editing.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      })
      setEditing(null)
      setShowForm(false)
      fetchBooks()
    } catch {
      setError('Failed to update book')
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this book?')) return
    try {
      await fetch(`${API_BASE}/books/${id}`, { method: 'DELETE' })
      fetchBooks()
    } catch {
      setError('Failed to delete book')
    }
  }

  return (
    <div>
      <div style={{ marginBottom: '20px' }}>
        <button onClick={() => { setEditing(null); setShowForm(true) }}>
          Add Book
        </button>
        <button onClick={fetchBooks} style={{ marginLeft: '10px' }}>
          Refresh
        </button>
      </div>

      {error && <p style={{ color: 'red' }}>{error}</p>}

      {showForm ? (
        <BookForm
          book={editing}
          onSubmit={editing ? handleUpdate : handleCreate}
          onCancel={() => { setShowForm(false); setEditing(null) }}
        />
      ) : loading ? (
        <p>Loading...</p>
      ) : (
        <BookList
          books={books}
          onEdit={(book) => { setEditing(book); setShowForm(true) }}
          onDelete={handleDelete}
        />
      )}
    </div>
  )
}
