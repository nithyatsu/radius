extension radius
extension testresources
extension kubernetes with {
  kubeConfig: ''
  namespace: 'udttoudtapp'
} as kubernetes
param registry string

param version string

@description('Specifies the port the container listens on.')
param port int = 8080

resource udttoudtenv1 'Applications.Core/environments@2023-10-01-preview' = {
  name: 'udttoudtenv1'
  location: 'global'
  properties: {
    compute: {
      kind: 'kubernetes'
      resourceId: 'self'
      namespace: 'udttoudtenv1'
    }
    recipes: {
      'Test.Resources/userTypeAlpha': {
        default: {
          templateKind: 'bicep'
          templatePath: '${registry}/test/testrecipes/test-bicep-recipes/dynamicrp_recipe:${version}'
          parameters: {
            port: port
          }
        }
      }
    }
  }
}

resource udttoudtapp1 'Applications.Core/applications@2023-10-01-preview' = {
  name: 'udttoudtapp1'
  location: 'global'
  properties: {
    environment: udttoudtenv1.id
  }
 }


resource udttoudtparent1 'Test.Resources/userTypeAlpha@2023-10-01-preview' = {
    name: 'udttoudtparent'
    properties: {
      environment: udttoudtenv1.id
      application: udttoudtapp1.id
      connections: {
      externalresource: {
        source: udttoudtchild1.id
      }
    }
  } 
}


resource udttoudtchild1 'Test.Resources/externalResource@2023-10-01-preview' = {
  name: 'udttoudtchild'
  location: 'global'
  properties: {
    environment: udttoudtenv1.id
    application: udttoudtapp1.id
    configMap: '{"app1.sample.properties":"property1=value1\\nproperty2=value2","app2.sample.properties":"property3=value3\\nproperty4=value4"}'
  }
}
