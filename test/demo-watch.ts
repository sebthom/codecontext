// Demo file for testing file watcher functionality
import { useState, useEffect } from 'react';

interface User {
  id: number;
  name: string;
  email: string;
}

class UserService {
  private users: User[] = [];

  constructor() {
    this.users = [];
  }

  async getUser(id: number): Promise<User | null> {
    return this.users.find(user => user.id === id) || null;
  }

  async addUser(user: User): Promise<void> {
    this.users.push(user);
  }

  async deleteUser(id: number): Promise<boolean> {
    const index = this.users.findIndex(user => user.id === id);
    if (index > -1) {
      this.users.splice(index, 1);
      return true;
    }
    return false;
  }
}

export default UserService;

// This file can be modified to test the file watcher
export const DEFAULT_TIMEOUT = 5000;