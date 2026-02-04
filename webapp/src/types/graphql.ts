import type { TypedDocumentNode as DocumentNode } from '@graphql-typed-document-node/core';
export type Maybe<T> = T | undefined;
export type InputMaybe<T> = T | undefined;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
export type MakeEmpty<T extends { [key: string]: unknown }, K extends keyof T> = { [_ in K]?: never };
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: { input: string; output: string; }
  String: { input: string; output: string; }
  Boolean: { input: boolean; output: boolean; }
  Int: { input: number; output: number; }
  Float: { input: number; output: number; }
  Date: { input: any; output: any; }
  Time: { input: any; output: any; }
  UUID: { input: any; output: any; }
};

export type DateRange = {
  DueAfter: Scalars['Date']['input'];
  DueBefore: Scalars['Date']['input'];
};

export type Mutation = {
  __typename?: 'Mutation';
  deleteTodo: Scalars['Boolean']['output'];
  updateTodo: Todo;
};


export type MutationDeleteTodoArgs = {
  id: Scalars['UUID']['input'];
};


export type MutationUpdateTodoArgs = {
  params: UpdateTodoParams;
};

export type Query = {
  __typename?: 'Query';
  listTodos: TodoPage;
};


export type QueryListTodosArgs = {
  dateRange: InputMaybe<DateRange>;
  page?: Scalars['Int']['input'];
  pageSize?: Scalars['Int']['input'];
  query: InputMaybe<Scalars['String']['input']>;
  sortBy: InputMaybe<TodoSortBy>;
  status: InputMaybe<TodoStatus>;
};

export type Todo = {
  __typename?: 'Todo';
  created_at: Scalars['Time']['output'];
  due_date: Scalars['Date']['output'];
  id: Scalars['UUID']['output'];
  status: TodoStatus;
  title: Scalars['String']['output'];
  updated_at: Scalars['Time']['output'];
};

export type TodoPage = {
  __typename?: 'TodoPage';
  items: Array<Todo>;
  nextPage: Maybe<Scalars['Int']['output']>;
  page: Scalars['Int']['output'];
  previousPage: Maybe<Scalars['Int']['output']>;
};

export type TodoSortBy =
  | 'createdAtAsc'
  | 'createdAtDesc'
  | 'dueDateAsc'
  | 'dueDateDesc';

export type TodoStatus =
  | 'DONE'
  | 'OPEN';

export type UpdateTodoParams = {
  due_date: InputMaybe<Scalars['Date']['input']>;
  id: Scalars['UUID']['input'];
  status: InputMaybe<TodoStatus>;
  title: InputMaybe<Scalars['String']['input']>;
};

export type ListTodosQueryVariables = Exact<{
  status: InputMaybe<TodoStatus>;
  query: InputMaybe<Scalars['String']['input']>;
  page: Scalars['Int']['input'];
  pageSize: Scalars['Int']['input'];
  dateRange: InputMaybe<DateRange>;
  sortBy: InputMaybe<TodoSortBy>;
}>;


export type ListTodosQuery = { __typename?: 'Query', listTodos: { __typename?: 'TodoPage', page: number, nextPage: number | undefined, previousPage: number | undefined, items: Array<{ __typename?: 'Todo', id: any, title: string, status: TodoStatus, due_date: any, created_at: any, updated_at: any }> } };


export const ListTodosDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"ListTodos"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"status"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"TodoStatus"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"query"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"dateRange"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"DateRange"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"sortBy"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"TodoSortBy"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"listTodos"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"status"},"value":{"kind":"Variable","name":{"kind":"Name","value":"status"}}},{"kind":"Argument","name":{"kind":"Name","value":"query"},"value":{"kind":"Variable","name":{"kind":"Name","value":"query"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}},{"kind":"Argument","name":{"kind":"Name","value":"pageSize"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}}},{"kind":"Argument","name":{"kind":"Name","value":"dateRange"},"value":{"kind":"Variable","name":{"kind":"Name","value":"dateRange"}}},{"kind":"Argument","name":{"kind":"Name","value":"sortBy"},"value":{"kind":"Variable","name":{"kind":"Name","value":"sortBy"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"due_date"}},{"kind":"Field","name":{"kind":"Name","value":"created_at"}},{"kind":"Field","name":{"kind":"Name","value":"updated_at"}}]}},{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"nextPage"}},{"kind":"Field","name":{"kind":"Name","value":"previousPage"}}]}}]}}]} as unknown as DocumentNode<ListTodosQuery, ListTodosQueryVariables>;