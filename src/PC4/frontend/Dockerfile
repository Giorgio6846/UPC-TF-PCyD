# Imagen base con Node
FROM node:20-alpine

# Directorio de trabajo
WORKDIR /app

# Copiamos solo los archivos de dependencias primero
COPY package.json package-lock.json* ./

# Instalamos dependencias
RUN npm install

# Copiamos el resto del código
COPY . .

# Puerto por defecto de Vite
EXPOSE 5173

# Comando por defecto (modo desarrollo)
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0", "--port", "5173"]
