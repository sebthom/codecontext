// Sample TypeScript file for testing real parsing
import { Component } from 'react';
import * as fs from 'fs';

interface User {
  id: number;
  name: string;
  email?: string;
}

class UserService {
  private users: User[] = [];

  constructor() {
    this.loadUsers();
  }

  public async getUser(id: number): Promise<User | null> {
    return this.users.find(user => user.id === id) || null;
  }

  public addUser(user: User): void {
    this.users.push(user);
  }

  private loadUsers(): void {
    // Load users from storage
    console.log('Loading users...');
  }
}

export default UserService;
export { User };

function processUsers(users: User[]): void {
  users.forEach(user => {
    console.log(`Processing user: ${user.name}`);
  });
}

const DEFAULT_TIMEOUT = 5000;
let currentUser: User | null = null;