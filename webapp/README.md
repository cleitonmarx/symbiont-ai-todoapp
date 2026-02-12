# Todo App Demo UI

This project is a developer-friendly UI for the Todo App Demo API. It is built using React and TypeScript, and it provides a modern interface for managing todos, including creating, updating, deleting, batch operations, and more. The application also displays an AI-generated board summary and features a real-time AI chat assistant.

## Getting Started

To run the application locally, follow these steps:

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd examples/todoapp/webapp
   ```

2. **Install dependencies**:
   ```bash
   npm install
   ```

3. **Set the API base URL** (optional):
   The API base URL is configured via the `VITE_API_BASE_URL` environment variable. By default, it points to `http://localhost:8080`.

4. **Run the application**:
   ```bash
   npm run dev
   ```

5. **Open your browser**:
   Navigate to `http://localhost:5173` to view the application.

## Features

- **Create Todos**: Add new todos with titles and due dates.
- **Edit & Complete Todos**: Rename or mark todos as done.
- **Batch Operations**: Select multiple todos and perform batch actions:
  - Change due dates
  - Mark as done
  - Delete
- **Batch Selection**: Use the header checkbox to select/deselect all visible todos.
- **Pagination**: Navigate through pages of todos.
- **Status Filter**: Filter todos by "Open" or "Done".
- **AI-Generated Board Summary**: Get insights into your todo board, including:
  - Total open/completed todos
  - Next up items with reasons
  - Overdue and near-deadline tasks
- **AI Chat Assistant**: Movable chat toggle button opens a real-time AI chat (using SSE for streaming responses).

## Demo Script

To test the application, follow these steps:

1. Create a new todo by entering a title and due date in the form and clicking "Add Todo".
2. View the list of todos and their statuses.
3. Update a todo by clicking the "Complete" or "Rename" buttons next to each todo.
4. Select multiple todos and perform batch actions (change due date, mark as done, or delete).
5. Use the filter to view only open or done todos.
6. Check the Board Summary at the top of the list for AI-generated insights.
7. Use the chat button for AI-powered, real-time help.

## License

This project is licensed under the MIT License.