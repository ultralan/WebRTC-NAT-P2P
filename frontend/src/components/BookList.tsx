import { Book } from '../types/book'

interface Props {
  books: Book[]
  onEdit: (book: Book) => void
  onDelete: (id: number) => void
}

export function BookList({ books, onEdit, onDelete }: Props) {
  if (books.length === 0) {
    return <p>No books found</p>
  }

  return (
    <table style={{ width: '100%', borderCollapse: 'collapse' }}>
      <thead>
        <tr style={{ borderBottom: '2px solid #333' }}>
          <th style={{ padding: '8px', textAlign: 'left' }}>Title</th>
          <th style={{ padding: '8px', textAlign: 'left' }}>Author</th>
          <th style={{ padding: '8px', textAlign: 'left' }}>ISBN</th>
          <th style={{ padding: '8px', textAlign: 'left' }}>Year</th>
          <th style={{ padding: '8px', textAlign: 'left' }}>Stock</th>
          <th style={{ padding: '8px', textAlign: 'left' }}>Actions</th>
        </tr>
      </thead>
      <tbody>
        {books.map((book) => (
          <tr key={book.id} style={{ borderBottom: '1px solid #ddd' }}>
            <td style={{ padding: '8px' }}>{book.title}</td>
            <td style={{ padding: '8px' }}>{book.author}</td>
            <td style={{ padding: '8px' }}>{book.isbn}</td>
            <td style={{ padding: '8px' }}>{book.year}</td>
            <td style={{ padding: '8px' }}>{book.stock}</td>
            <td style={{ padding: '8px' }}>
              <button onClick={() => onEdit(book)} style={{ marginRight: '8px' }}>
                Edit
              </button>
              <button onClick={() => onDelete(book.id)} style={{ color: 'red' }}>
                Delete
              </button>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
