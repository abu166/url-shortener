# Use official Node.js image as the base
FROM node:18-alpine

# Set working directory
WORKDIR /app

# Copy package.json and package-lock.json
COPY package.json ./
COPY package-lock.json* ./

# Install dependencies
RUN npm install

# Copy the rest of the application
COPY . .

# Set environment variable for build
ARG REACT_APP_API_URL
ENV REACT_APP_API_URL=http://localhost:8080

# Build the React app
RUN npm run build

# Serve the build files
CMD ["npm", "run", "serve"]