class IMClient {
    constructor() {
        this.socket = null;
        this.currentUser = null;
        this.currentRoom = 'general'; // 默认房间
        this.deviceInfo = this.getDeviceInfo(); // 获取设备信息
        this.deviceCount = 0; // 当前用户的设备数量
        this.otherDevices = []; // 用户的其他设备列表
        this.typingTimeout = null;
        this.onlineUsers = new Set();
        
        this.initializeElements();
        this.setupEventListeners();
        this.updateUI();
    }

    initializeElements() {
        // Login elements
        this.usernameInput = document.getElementById('username') || document.getElementById('usernameInput');
        this.loginButton = document.getElementById('loginBtn') || document.getElementById('loginButton');
        this.logoutButton = document.getElementById('logoutButton');
        this.mainContent = document.getElementById('mainContent');
        
        // Chat elements
        this.messagesContainer = document.getElementById('messages') || document.getElementById('messagesContainer');
        this.messageInput = document.getElementById('messageInput');
        this.sendButton = document.getElementById('sendBtn') || document.getElementById('sendButton');
        this.fileInput = document.getElementById('fileInput');
        this.chatTitle = document.getElementById('chatTitle');
        this.statusIndicator = document.getElementById('statusIndicator');
        this.typingIndicator = document.getElementById('typingIndicator');
        
        // Room elements (可选元素，用于debug页面)
        this.roomInput = document.getElementById('roomInput');
        this.joinRoomButton = document.getElementById('joinRoomButton');
        this.roomList = document.getElementById('roomList');
        this.userList = document.getElementById('userList');
    }

    setupEventListeners() {
        // Login
        if (this.loginButton) {
            this.loginButton.addEventListener('click', () => this.login());
        }
        if (this.logoutButton) {
            this.logoutButton.addEventListener('click', () => this.logout());
        }
        if (this.usernameInput) {
            this.usernameInput.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') this.login();
            });
        }

        // Messaging events will be set up after chat container is shown

        // Room management (仅在debug页面存在)
        if (this.joinRoomButton) {
            this.joinRoomButton.addEventListener('click', () => this.joinCustomRoom());
        }
        if (this.roomInput) {
            this.roomInput.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') this.joinCustomRoom();
            });
        }

        // Room list events will be set up after chat container is shown
    }

    setupChatEventListeners() {
        console.log('Setting up chat event listeners...');
        
        // Re-initialize chat elements since they're now visible
        this.messagesContainer = document.getElementById('messages');
        this.messageInput = document.getElementById('messageInput');
        this.sendButton = document.getElementById('sendBtn');
        this.fileInput = document.getElementById('fileInput');
        this.typingIndicator = document.getElementById('typingIndicator');
        
        // Form submit event
        const messageForm = document.getElementById('messageForm');
        if (messageForm) {
            console.log('Form found, adding submit listener');
            messageForm.addEventListener('submit', (e) => {
                console.log('Form submit event triggered');
                e.preventDefault(); // 阻止表单默认提交行为
                this.sendMessage();
            });
        } else {
            console.log('Form not found!');
        }
        
        // Send button click event
        if (this.sendButton) {
            console.log('Send button found, adding click listener');
            this.sendButton.addEventListener('click', (e) => {
                console.log('Send button clicked');
                e.preventDefault(); // 确保不会触发默认行为
                this.sendMessage();
            });
        } else {
            console.log('Send button not found!');
        }
        
        // Message input events
        if (this.messageInput) {
            this.messageInput.addEventListener('keypress', (e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault();
                    this.sendMessage();
                }
            });

            // Typing indicator
            this.messageInput.addEventListener('input', () => this.handleTyping());
            this.messageInput.addEventListener('keypress', () => this.handleTyping());

            // Auto-resize textarea
            this.messageInput.addEventListener('input', () => this.autoResizeTextarea());
        }

        // File upload
        if (this.fileInput) {
            this.fileInput.addEventListener('change', (e) => this.handleFileUpload(e));
        }
        
        // Upload button
        const uploadBtn = document.getElementById('uploadBtn');
        if (uploadBtn) {
            uploadBtn.addEventListener('click', (e) => {
                e.preventDefault(); // 确保不会触发表单提交
                if (this.fileInput) {
                    this.fileInput.click();
                }
            });
        }
        
        // Room list events
        const roomItems = document.querySelectorAll('.room-item');
        roomItems.forEach(item => {
            item.addEventListener('click', () => {
                const roomId = item.dataset.room;
                if (roomId) {
                    this.switchRoom(roomId);
                }
            });
        });
        
        // Scroll to bottom button
        const scrollToBottomBtn = document.getElementById('scrollToBottomBtn');
        if (scrollToBottomBtn) {
            scrollToBottomBtn.addEventListener('click', () => {
                this.scrollToBottom();
                scrollToBottomBtn.classList.remove('show');
            });
        }
        
        // Monitor scroll position to show/hide scroll-to-bottom button
        if (this.messagesContainer) {
            this.messagesContainer.addEventListener('scroll', () => {
                this.handleScroll();
            });
        }
        
        console.log('Chat event listeners setup complete');
    }

    updateUI() {
        const isLoggedIn = this.currentUser !== null;
        
        if (this.mainContent) {
            this.mainContent.classList.toggle('hidden', !isLoggedIn);
        }
        if (this.loginButton) {
            this.loginButton.classList.toggle('hidden', isLoggedIn);
        }
        if (this.logoutButton) {
            this.logoutButton.classList.toggle('hidden', !isLoggedIn);
        }
        if (this.sendButton) {
            this.sendButton.disabled = !isLoggedIn;
        }
        if (this.messageInput) {
            this.messageInput.disabled = !isLoggedIn;
        }
    }

    login() {
        const username = this.usernameInput.value.trim();
        if (!username) {
            alert('请输入用户名');
            return;
        }

        this.currentUser = {
            id: username + '_' + Date.now(),
            name: username
        };

        this.connectSocket();
        this.updateUI();
    }

    logout() {
        if (this.socket) {
            this.socket.disconnect();
            this.socket = null;
        }
        this.currentUser = null;
        this.onlineUsers.clear();
        this.updateUI();
        this.updateConnectionStatus('disconnected');
        this.clearMessages();
        this.clearUserList();
    }

    connectSocket() {
        this.updateConnectionStatus('connecting');
        
        this.socket = io('/', {
            transports: ['websocket', 'polling'],
            upgrade: true,
            rememberUpgrade: true
        });

        this.setupSocketListeners();
        
        this.socket.on('connect', () => {
            console.log('Connected to server');
            this.updateConnectionStatus('connected');
            
            // Join as user
            this.socket.emit('join', {
                userId: this.currentUser.id,
                userName: this.currentUser.name,
                avatar: '',
                deviceInfo: this.deviceInfo,
                deviceCount: this.deviceCount
            });
            
            // Join default room
            this.joinRoom(this.currentRoom);
        });

        this.socket.on('disconnect', (reason) => {
            console.log('Disconnected:', reason);
            this.updateConnectionStatus('disconnected');
            this.clearUserList();
            this.deviceCount = 0;
            this.otherDevices = [];
            this.updateDeviceInfo();
        });

        this.socket.on('connect_error', (error) => {
            console.error('Connection error:', error);
            this.updateConnectionStatus('disconnected');
            this.showSystemMessage('连接服务器失败，请检查网络连接');
        });
    }

    setupSocketListeners() {
        // Join confirmation
        this.socket.on('joined', (data) => {
            console.log('Joined successfully:', data);
            // 保持currentUser的对象结构
            if (typeof this.currentUser === 'object') {
                this.currentUser.name = data.userName;
            } else {
                this.currentUser = {
                    id: data.userId || data.userName,
                    name: data.userName
                };
            }
            this.deviceCount = data.deviceCount || 1;
            this.updateStatus(`已加入 (${this.deviceCount} 设备在线)`, 'joined');
            this.updateDeviceInfo();
            document.getElementById('loginForm').style.display = 'none';
            document.getElementById('chatContainer').style.display = 'block';
            this.setupChatEventListeners(); // 在显示聊天界面后绑定事件
            this.updateUserInfo(data);
        });

        // Message received
        this.socket.on('message', (message) => {
            this.displayMessage(message);
        });

        // User status updates
        this.socket.on('user_status', (data) => {
            this.updateUserStatus(data.userName, data.status);
        });

        // Room events
        this.socket.on('room_joined', (data) => {
            console.log('Room joined:', data);
            this.currentRoom = data.roomId;
            this.updateRoomInfo(data.roomId);
        });

        this.socket.on('user_joined_room', (data) => {
            this.showSystemMessage(`${data.userName} 加入了房间`);
        });

        this.socket.on('user_left_room', (data) => {
            this.showSystemMessage(`${data.userName} 离开了房间`);
        });

        // Typing events
        this.socket.on('typing', (data) => {
            this.showTypingIndicator(data.userName);
        });

        this.socket.on('stop_typing', (data) => {
            this.hideTypingIndicator(data.userName);
        });

        // Error handling
        this.socket.on('error', (data) => {
            console.error('Socket error:', data);
            this.showSystemMessage(`错误: ${data.message}`);
        });

        // Multi-device events
        this.socket.on('device_connected', (data) => {
            console.log('New device connected:', data);
            this.deviceCount = data.deviceCount;
            this.addDeviceToList(data);
            this.showNotification(`新设备已连接: ${data.deviceInfo}`, 'info');
            this.updateDeviceInfo();
        });

        this.socket.on('device_disconnected', (data) => {
            console.log('Device disconnected:', data);
            this.deviceCount = data.deviceCount;
            this.removeDeviceFromList(data.sessionId);
            this.showNotification(`设备已断开连接`, 'warning');
            this.updateDeviceInfo();
        });
    }

    sendMessage() {
        const content = this.messageInput.value.trim();
        if (!content || !this.socket || !this.currentUser) return;

        const messageData = {
            type: 'text',
            content: content,
            sender: this.currentUser.name || this.currentUser,
            roomId: this.currentRoom,
            timestamp: new Date().toISOString()
        };

        this.socket.emit('message', messageData);
        this.messageInput.value = '';
        this.autoResizeTextarea();
        this.stopTyping();
        
        // 发送消息后强制滚动到底部
        setTimeout(() => {
            this.scrollToBottom();
        }, 50);
    }

    handleFileUpload(event) {
        const file = event.target.files[0];
        if (!file || !this.socket || !this.currentUser) return;

        // Check file size (10MB limit)
        if (file.size > 10 * 1024 * 1024) {
            alert('文件大小不能超过10MB');
            return;
        }

        const reader = new FileReader();
        reader.onload = (e) => {
            const fileData = {
                fileName: file.name,
                fileData: e.target.result.split(',')[1], // Remove data:type;base64, prefix
                fileType: file.type || 'application/octet-stream',
                sender: this.currentUser.name || this.currentUser,
                roomId: this.currentRoom
            };

            this.socket.emit('file_upload', fileData);
        };

        reader.readAsDataURL(file);
        event.target.value = ''; // Clear file input
    }

    displayMessage(message) {
        const messageElement = document.createElement('div');
        messageElement.className = 'message';
        
        const isOwnMessage = this.currentUser && message.sender === (this.currentUser.name || this.currentUser);
        messageElement.classList.add(isOwnMessage ? 'own' : 'other');
        
        if (message.type === 'file') {
            messageElement.classList.add('file');
        }

        const timestamp = new Date(message.timestamp).toLocaleTimeString();
        const senderName = isOwnMessage ? '你' : message.sender;

        let content = '';
        if (message.type === 'file' && message.metadata) {
            const metadata = message.metadata;
            content = `
                <div class="message-header">${senderName} • ${timestamp}</div>
                <div class="message-content">
                    <div class="file-info">
                        <div class="file-icon">📎</div>
                        <div class="file-details">
                            <div class="file-name">${metadata.fileName}</div>
                            <div class="file-size">${this.formatFileSize(metadata.fileSize)}</div>
                        </div>
                        <a href="${metadata.fileURL}" target="_blank" download="${metadata.fileName}">
                            <button style="padding: 5px 10px; background: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer;">下载</button>
                        </a>
                    </div>
                </div>
            `;
        } else {
            content = `
                <div class="message-header">${senderName} • ${timestamp}</div>
                <div class="message-content">${this.escapeHtml(message.content)}</div>
            `;
        }

        messageElement.innerHTML = content;
        if (this.messagesContainer) {
            this.messagesContainer.appendChild(messageElement);
        }
        this.smartScrollToBottom();
    }

    showSystemMessage(content) {
        const messageElement = document.createElement('div');
        messageElement.className = 'message system';
        messageElement.innerHTML = `<div class="message-content">${this.escapeHtml(content)}</div>`;
        if (this.messagesContainer) {
            this.messagesContainer.appendChild(messageElement);
        }
        this.smartScrollToBottom();
    }

    joinRoom(roomId) {
        if (!this.socket || !this.currentUser) return;
        
        this.socket.emit('join_room', {
            roomId: roomId,
            userName: this.currentUser.name || this.currentUser
        });
    }

    leaveRoom(roomId) {
        if (!this.socket || !this.currentUser) return;
        
        this.socket.emit('leave_room', {
            roomId: roomId,
            userName: this.currentUser.name || this.currentUser
        });
    }

    switchRoom(roomId) {
        if (this.currentRoom === roomId) return;
        
        // Leave current room
        if (this.currentRoom) {
            this.leaveRoom(this.currentRoom);
        }
        
        // Join new room
        this.currentRoom = roomId;
        this.joinRoom(roomId);
        
        // Update UI
        this.updateRoomTitle(roomId);
        this.updateActiveRoom(roomId);
        this.clearMessages();
        this.showSystemMessage(`已切换到 ${this.getRoomDisplayName(roomId)}`);
        
        // 切换房间后滚动到底部
        setTimeout(() => {
            this.scrollToBottom();
        }, 100);
    }

    joinCustomRoom() {
        if (!this.roomInput) return;
        
        const roomName = this.roomInput.value.trim();
        if (!roomName) return;
        
        this.addCustomRoom(roomName);
        this.switchRoom(roomName);
        this.roomInput.value = '';
    }

    addCustomRoom(roomId) {
        if (!this.roomList) return;
        
        // Check if room already exists
        const existingRoom = this.roomList.querySelector(`[data-room="${roomId}"]`);
        if (existingRoom) return;
        
        const roomElement = document.createElement('li');
        roomElement.dataset.room = roomId;
        roomElement.textContent = `🏠 ${roomId}`;
        roomElement.addEventListener('click', () => this.switchRoom(roomId));
        this.roomList.appendChild(roomElement);
    }

    updateRoomTitle(roomId) {
        if (this.chatTitle) {
            this.chatTitle.textContent = this.getRoomDisplayName(roomId);
        }
    }

    updateActiveRoom(roomId) {
        if (this.roomList) {
            this.roomList.querySelectorAll('li').forEach(li => {
                li.classList.toggle('active', li.dataset.room === roomId);
            });
        }
    }

    getRoomDisplayName(roomId) {
        const roomMap = {
            'general': '📍 通用聊天室',
            'tech': '💻 技术讨论',
            'random': '🎲 随机聊天'
        };
        return roomMap[roomId] || `🏠 ${roomId}`;
    }

    handleTyping() {
        if (!this.socket || !this.currentUser) return;
        
        // Send typing event
        this.socket.emit('typing', {
            userName: this.currentUser.name || this.currentUser,
            roomId: this.currentRoom
        });
        
        // Clear existing timeout
        if (this.typingTimeout) {
            clearTimeout(this.typingTimeout);
        }
        
        // Set timeout to stop typing
        this.typingTimeout = setTimeout(() => {
            this.stopTyping();
        }, 2000);
    }

    stopTyping() {
        if (!this.socket || !this.currentUser) return;
        
        this.socket.emit('stop_typing', {
            userName: this.currentUser.name || this.currentUser,
            roomId: this.currentRoom
        });
    }

    showTypingIndicator(userName) {
        if (!this.currentUser || userName === this.currentUser.name) return;
        
        if (this.typingIndicator) {
            this.typingIndicator.textContent = `${userName} 正在输入...`;
            this.typingIndicator.classList.remove('hidden');
        }
    }

    hideTypingIndicator(userName) {
        if (!this.currentUser || userName === this.currentUser.name) return;
        
        if (this.typingIndicator) {
            this.typingIndicator.classList.add('hidden');
        }
    }

    updateUserStatus(userName, status) {
        if (status === 'online') {
            this.onlineUsers.add(userName);
        } else {
            this.onlineUsers.delete(userName);
        }
        this.updateUserList();
    }

    updateUserList() {
        if (this.userList) {
            this.userList.innerHTML = '';
            this.onlineUsers.forEach(userName => {
                const userElement = document.createElement('li');
                userElement.innerHTML = `
                    <span class="user-status ${this.currentUser && userName === this.currentUser.name ? 'online' : 'online'}"></span>
                    ${userName}
                `;
                this.userList.appendChild(userElement);
            });
        }
    }

    clearUserList() {
        if (this.userList) {
            this.userList.innerHTML = '';
        }
        this.onlineUsers.clear();
    }

    updateConnectionStatus(status) {
        if (this.statusIndicator) {
            this.statusIndicator.className = `status-indicator ${status}`;
            const statusText = {
                'connecting': '连接中...',
                'connected': '已连接',
                'disconnected': '已断开'
            };
            this.statusIndicator.textContent = statusText[status] || status;
        }
    }

    autoResizeTextarea() {
        const textarea = this.messageInput;
        if (textarea) {
            textarea.style.height = 'auto';
            textarea.style.height = Math.min(textarea.scrollHeight, 100) + 'px';
        }
    }

    scrollToBottom() {
        if (this.messagesContainer) {
            // 使用 setTimeout 确保 DOM 更新后再滚动
            setTimeout(() => {
                this.messagesContainer.scrollTop = this.messagesContainer.scrollHeight;
            }, 10);
        }
    }

    // 检查用户是否在底部附近，如果是则自动滚动
    shouldAutoScroll() {
        if (!this.messagesContainer) return true;
        
        const threshold = 100; // 距离底部100px内认为是在底部
        const isNearBottom = this.messagesContainer.scrollHeight - this.messagesContainer.scrollTop - this.messagesContainer.clientHeight < threshold;
        return isNearBottom;
    }

    // 智能滚动：只有当用户在底部附近时才自动滚动
    smartScrollToBottom() {
        if (this.shouldAutoScroll()) {
            this.scrollToBottom();
        }
    }

    // 处理滚动事件，显示/隐藏滚动到底部按钮
    handleScroll() {
        const scrollToBottomBtn = document.getElementById('scrollToBottomBtn');
        if (!scrollToBottomBtn || !this.messagesContainer) return;
        
        const isAtBottom = this.shouldAutoScroll();
        
        if (isAtBottom) {
            scrollToBottomBtn.classList.remove('show');
        } else {
            scrollToBottomBtn.classList.add('show');
        }
    }

    clearMessages() {
        if (this.messagesContainer) {
            this.messagesContainer.innerHTML = '';
        }
    }

    formatFileSize(bytes) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Additional methods for multi-device support
    getDeviceInfo() {
        const ua = navigator.userAgent;
        let deviceType = 'Desktop';
        let browser = 'Unknown';
        
        // 检测设备类型
        if (/Mobile|Android|iPhone|iPad/.test(ua)) {
            if (/iPad/.test(ua)) {
                deviceType = 'iPad';
            } else if (/iPhone/.test(ua)) {
                deviceType = 'iPhone';
            } else if (/Android/.test(ua)) {
                deviceType = 'Android';
            } else {
                deviceType = 'Mobile';
            }
        }
        
        // 检测浏览器
        if (ua.includes('Chrome')) browser = 'Chrome';
        else if (ua.includes('Firefox')) browser = 'Firefox';
        else if (ua.includes('Safari')) browser = 'Safari';
        else if (ua.includes('Edge')) browser = 'Edge';
        
        const timestamp = new Date().toLocaleTimeString();
        return `${deviceType} - ${browser} (${timestamp})`;
    }

    updateDeviceInfo() {
        const deviceInfoElement = document.getElementById('deviceInfo');
        if (deviceInfoElement) {
            deviceInfoElement.innerHTML = `
                <div class="device-status">
                    <strong>设备状态:</strong>
                    <span class="device-count">${this.deviceCount} 设备在线</span>
                </div>
                <div class="current-device">
                    <strong>当前设备:</strong> ${this.escapeHtml(this.deviceInfo)}
                </div>
            `;
        }
    }

    addDeviceToList(data) {
        const device = {
            sessionId: data.sessionId,
            deviceInfo: data.deviceInfo
        };
        this.otherDevices.push(device);
    }

    removeDeviceFromList(sessionId) {
        this.otherDevices = this.otherDevices.filter(device => device.sessionId !== sessionId);
    }

    updateRoomInfo(roomId) {
        const roomInfoElement = document.querySelector('.room-info');
        if (roomInfoElement) {
            roomInfoElement.textContent = `当前房间: ${roomId}`;
        }
    }

    updateUserInfo(data) {
        const userInfoElement = document.querySelector('.user-info');
        if (userInfoElement) {
            userInfoElement.innerHTML = `
                <span>👤 ${this.escapeHtml(data.userName)}</span>
                <span class="device-info">设备: ${this.escapeHtml(data.deviceInfo)}</span>
            `;
        }
    }

    showNotification(message, type = 'info') {
        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.textContent = message;

        document.body.appendChild(notification);

        setTimeout(() => {
            notification.classList.add('show');
        }, 100);

        setTimeout(() => {
            notification.classList.remove('show');
            setTimeout(() => {
                document.body.removeChild(notification);
            }, 300);
        }, 3000);
    }

    updateStatus(message, className = '') {
        const statusElement = document.getElementById('statusIndicator');
        statusElement.textContent = message;
        statusElement.className = `status-indicator ${className}`;
    }
}

// Initialize the IM client when the page loads
document.addEventListener('DOMContentLoaded', () => {
    new IMClient();
}); 