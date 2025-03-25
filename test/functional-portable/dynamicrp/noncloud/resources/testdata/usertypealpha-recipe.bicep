extension testresources
extension radius

param registry string

param version string

@description('Specifies the location for resources.')
param location string = 'global'

resource env 'Applications.Core/environments@2023-10-01-preview' = {
  name: 'usertypealpha-recipe-env'
  location: location
  properties: {
    compute: {
      kind: 'kubernetes'
      resourceId: 'self'
      namespace: 'default-usertypealpha-recipe'
    }
    recipes: {
      'Test.Resources/userTypeAlpha': {
        default: {
          templateKind: 'bicep'
          templatePath: 'lakacr2.azurecr.io/usertypealpha:latest'          
        }
      }
    }
  }
}

resource app 'Applications.Core/applications@2023-10-01-preview' = {
  name: 'usertypealpha-recipe-app'
  location: location
  properties: {
    environment: env.id
  }
}

resource usertypealpha1 'Test.Resources/userTypeAlpha@2023-10-01-preview' = {
  name: 'usertypealpha123'
  location: location
  properties: {
    application: app.id
    environment: env.id
  }
}
