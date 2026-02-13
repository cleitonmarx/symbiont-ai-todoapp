import axios from 'axios';
import {
  ListTodosQuery,
  TodoSortBy,
  TodoStatus,
} from '../types/graphql';

const GRAPHQL_ENDPOINT = import.meta.env.VITE_GRAPHQL_ENDPOINT || 'http://localhost:8085/v1/query';

const LIST_TODOS_QUERY = `
  query ListTodos(
    $status: TodoStatus
    $search: String
    $searchType: SearchType
    $page: Int!
    $pageSize: Int!
    $dateRange: DateRange
    $sortBy: TodoSortBy
  ) {
    listTodos(
      status: $status
      search: $search
      searchType: $searchType
      page: $page
      pageSize: $pageSize
      dateRange: $dateRange
      sortBy: $sortBy
    ) {
      items {
        id
        title
        status
        due_date
        created_at
        updated_at
      }
      page
      nextPage
      previousPage
    }
  }
`;

export interface GqlListTodosVariables {
  status?: TodoStatus;
  search?: string;
  searchType?: 'TITLE' | 'SIMILARITY';
  page: number;
  pageSize: number;
  dateRange?: { DueAfter: string; DueBefore: string };
  sortBy?: TodoSortBy;
}

export async function gqlListTodos(variables: GqlListTodosVariables) {
  const response = await axios.post<{ data: ListTodosQuery }>(
    GRAPHQL_ENDPOINT,
    {
      query: LIST_TODOS_QUERY,
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
