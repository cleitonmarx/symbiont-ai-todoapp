import axios from 'axios';
import { print } from 'graphql';
import {
  ListTodosDocument,
  ListTodosQuery,
  ListTodosQueryVariables,
  TodoStatus,
} from '../types/graphql';

const GRAPHQL_ENDPOINT = import.meta.env.VITE_GRAPHQL_ENDPOINT || 'http://localhost:8085/v1/query';

export async function gqlListTodos(variables: ListTodosQueryVariables) {
  const response = await axios.post<{ data: ListTodosQuery }>(
    GRAPHQL_ENDPOINT,
    {
      query: print(ListTodosDocument),
      variables,
    }
  );
  return response.data.data.listTodos;
}

export async function gqlBatchUpdateTodos(
  ids: string[],
  update: Partial<{ due_date: string; status: TodoStatus }>,
) {
  let mutation = 'mutation BatchUpdateTodos {';
  ids.forEach((id, idx) => {
    mutation += `
      update${idx}: updateTodo(params: { id: "${id}"${update.due_date ? `, due_date: "${update.due_date}"` : ''}${update.status ? `, status: ${update.status}` : ''} }) {
        id
      }
    `;
  });
  mutation += '}';
  await axios.post(GRAPHQL_ENDPOINT, { query: mutation });
}

export async function gqlBatchDeleteTodos(ids: string[]) {
  let mutation = 'mutation BatchDeleteTodos {';
  ids.forEach((id, idx) => {
    mutation += `
      delete${idx}: deleteTodo(id: "${id}")
    `;
  });
  mutation += '}';
  await axios.post(GRAPHQL_ENDPOINT, { query: mutation });
}
