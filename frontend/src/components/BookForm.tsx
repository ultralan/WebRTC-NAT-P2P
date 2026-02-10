import { useState, useEffect } from 'react'
import { Book } from '../types/book'

interface Props {
  book?: Book | null
  onSubmit: (data: Partial<Book>) => void
  onCancel: () => void
}

export function BookForm({ book, onSubmit, onCancel }: Props) {
  const [form, setForm] = useState({
    title: '',
    author: '',
    isbn: '',
    publisher: '',
    year: new Date().getFullYear(),
    stock: 0,
  })

  useEffect(() => {
    if (book) {
      setForm({
        title: book.title,
        author: book.author,
        isbn: book.isbn,
        publisher: book.publisher,
        year: book.year,
        stock: book.stock,
      })
    }
  }, [book])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSubmit(form)
  }

  const inputStyle = { width: '100%', padding: '8px', marginBottom: '10px' }

  return (
    <form onSubmit={handleSubmit} style={{ maxWidth: '400px' }}>
      <div>
        <label>Title *</label>
        <input
          style={inputStyle}
          value={form.title}
          onChange={(e) => setForm({ ...form, title: e.target.value })}
          required
        />
      </div>
      <div>
        <label>Author</label>
        <input
          style={inputStyle}
          value={form.author}
          onChange={(e) => setForm({ ...form, author: e.target.value })}
        />
      </div>
      <div>
        <label>ISBN</label>
        <input
          style={inputStyle}
          value={form.isbn}
          onChange={(e) => setForm({ ...form, isbn: e.target.value })}
        />
      </div>
      <div>
        <label>Publisher</label>
        <input
          style={inputStyle}
          value={form.publisher}
          onChange={(e) => setForm({ ...form, publisher: e.target.value })}
        />
      </div>
      <div>
        <label>Year</label>
        <input
          style={inputStyle}
          type="number"
          value={form.year}
          onChange={(e) => setForm({ ...form, year: parseInt(e.target.value) })}
        />
      </div>
      <div>
        <label>Stock</label>
        <input
          style={inputStyle}
          type="number"
          value={form.stock}
          onChange={(e) => setForm({ ...form, stock: parseInt(e.target.value) })}
        />
      </div>
      <div style={{ marginTop: '10px' }}>
        <button type="submit" style={{ marginRight: '10px' }}>
          {book ? 'Update' : 'Create'}
        </button>
        <button type="button" onClick={onCancel}>
          Cancel
        </button>
      </div>
    </form>
  )
}
