param registry string
param env string

resource app 'radius.dev/Application@v1alpha3' = {
  name: 'azure-cli-parameters'

  resource a 'Container' = {
    name: 'a'
    properties: {
      container: {
        image: '${registry}/magpie:latest'
        env: {
          COOL_SETTING: env
        }
      }
    }
  }

  resource b 'Container' = {
    name: 'b'
    properties: {
      container: {
        image: '${registry}/magpie:latest'
      }
    }
  }
}
