# Todo Completion Email Demo UI

This project is a developer-friendly UI for the Todo Completion Email Demo API. It is built using React and TypeScript, and it provides a simple interface for managing todos, including creating, updating, and listing them with their email delivery statuses. The application also displays an AI-generated board summary that provides insights into your todo list.

## Getting Started

To run the application locally, follow these steps:

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd webapp
   ```

2. **Install dependencies**:
   ```bash
   npm install
   ```

3. **Set the API base URL**:
   The API base URL is configured via the `VITE_API_BASE_URL` environment variable. By default, it points to `http://localhost:8080`.

4. **Run the application**:
   ```bash
   npm run dev
   ```

5. **Open your browser**:
   Navigate to `http://localhost:5173` to view the application.

## Features

- **Create Todos**: Add new todos with titles and due dates
- **Manage Todos**: Update todo titles or mark them as complete
- **Email Status Tracking**: Monitor the delivery status of completion emails
- **AI-Generated Board Summary**: Get insights into your todo board with AI-generated summaries, including:
  - Total count of open and completed todos
  - Next up items with reasons
  - Overdue tasks
  - Tasks near deadline

## Demo Script

To test the application, follow these steps:

1. Create a new todo by entering a title and due date in the form and clicking "Add Todo"
2. View the list of todos and their statuses
3. Update a todo by clicking the "Complete" or "Rename" buttons next to each todo
4. Observe the email delivery status updates as you complete todos
5. Check the Board Summary at the top of the list for AI-generated insights (if available)

## License

This project is licensed under the MIT License.